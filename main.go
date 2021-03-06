package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type webhook struct{}

func (vh *webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get webhook body with the admission review.
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		http.Error(w, "no body found", http.StatusBadRequest)
		return
	}

	ar := &admissionv1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, ar); err != nil {
		http.Error(w, "could not decode the admission review from the request", http.StatusBadRequest)
		return
	}
	fmt.Println("New request ", ar.Request.UID, ar.Request.Kind, ar.Request.Operation, ar.Request.Namespace)
	if ar.Request.Kind.Kind != "Pod" {
		fmt.Println("Resource type is not pod, skipped", ar.Request.Kind)
		return
	}

	status := "Success"
	pod := &corev1.Pod{}
	if _, _, err := deserializer.Decode(ar.Request.Object.Raw, nil, pod); err != nil {
		http.Error(w, "could not decode admission request object", http.StatusBadRequest)
		return
	}
	for _, container := range pod.Spec.Containers {
		limits := container.Resources.Limits
		requests := container.Resources.Requests

		cpuLimit := limits["cpu"]
		cpuRequest := requests["cpu"]
		fmt.Println("Container resources cpu ", cpuLimit, cpuRequest)

		limit, ok := cpuLimit.AsInt64()
		if !ok {
			continue
		}

		request, ok := cpuRequest.AsInt64()
		if !ok {
			continue
		}

		if request > 0 {
			ratio := limit / request
			fmt.Println("Container overcommit ratio ", container.Name, ratio)
			if ratio > 3 {
				status = "Failure"
			}
		}
	}
	allowed := true
	if status != "Success" {
		allowed = false
	}

	admissionResp := &admissionv1beta1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: allowed,
		Result: &metav1.Status{
			Status:  status,
			Message: "ok",
		},
	}

	// Forge the review response.
	aResponse := admissionv1beta1.AdmissionReview{
		Response: admissionResp,
	}
	resp, err := json.Marshal(aResponse)
	if err != nil {
		http.Error(w, "error marshaling to json admission review response", http.StatusInternalServerError)
		return
	}
	// Forge the HTTP response.
	// If the received admission review has failed mark the response as failed.
	if admissionResp.Result != nil && admissionResp.Result.Status == metav1.StatusFailure {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(resp); err != nil {
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func main() {
	srv := &http.Server{
		Addr:    ":8443",
		Handler: &webhook{},
		// TLSConfig: cfg,
	}
	fmt.Println("Start listen:8443")
	log.Fatal(srv.ListenAndServeTLS("certs/server.crt", "certs/server.key"))
}

// generate ssl certs

// openssl req \
//     -newkey rsa:2048 \
//     -nodes \
//     -days 3650 \
//     -x509 \
//     -keyout ca.key \
//     -out ca.crt \
//     -subj "/CN=*"

// openssl req \
//     -newkey rsa:2048 \
//     -nodes \
//     -keyout server.key \
//     -out server.csr \
//     -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=*"

// openssl x509 \
//     -req \
//     -days 365 \
//     -sha256 \
//     -in server.csr \
//     -CA ca.crt \
//     -CAkey ca.key \
//     -CAcreateserial \
//     -out server.crt \
//     -extfile <(echo subjectAltName = IP:127.0.0.1)
