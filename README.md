# Kubernetes Admission Webhook Example


 Kubernetes administrators would like to control the level of overcommit and manage container density on nodes, masters can be configured to override the ratio between request and limit set on developer containers.

We can use admission controller to modify/validate pod resource.


## Instructions


### Step 1/3 enable  admissionregistration.k8s.io/v1beta1=true & plugins

The api-server should add config below:


`--runtime-config: batch/v2alpha1=true, admissionregistration.k8s.io/v1beta1=true`

`--enable-admission-plugins: NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota`


### Step 2/3 Deploy webhook 

`kubectl apply -f webhook.yaml`

Here is an example yaml:


```
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: taoge.example.webhook
webhooks:
- name: taoge.example.webhook
  rules:
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
  clientConfig:
    url: https://10.8.0.172:8443
    caBundle: ""

```

Note that the `caBundle` is base64 encode of `pem` file.


### Step 3/3 Run webhook server

`go run main.go` 
will start a really simple http server. It justs receive k8s requests and return ok response.



## References

https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/


https://docs.okd.io/latest/architecture/additional_concepts/dynamic_admission_controllers.html


https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/



