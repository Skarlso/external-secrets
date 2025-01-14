## Google Cloud Secret Manager

External Secrets Operator integrates with [GCP Secret Manager](https://cloud.google.com/secret-manager) for secret management.

## Authentication

### Workload Identity

Your Google Kubernetes Engine (GKE) applications can consume GCP services like Secrets Manager without using static, long-lived authentication tokens. This is our recommended approach of handling credentials in GCP. ESO offers two options for integrating with GKE workload identity: **pod-based workload identity** and **using service accounts directly**. Before using either way you need to create a service account - this is covered below.

#### Creating Workload Identity Service Accounts

You can find the documentation for Workload Identity [here](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity). We will walk you through how to navigate it here.

Search [the document](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for this editable values and change them to your values:
_Note: If you have installed ESO, a serviceaccount has already been created. You can either patch the existing `external-secrets` SA or create a new one that fits your needs._

- `CLUSTER_NAME`: The name of your cluster
- `PROJECT_ID`: Your project ID (not your Project number nor your Project name)
- `K8S_NAMESPACE`: For us following these steps here it will be `es`, but this will be the namespace where you deployed the external-secrets operator
- `KSA_NAME`: external-secrets (if you are not creating a new one to attach to the deployment)
- `GSA_NAME`: external-secrets for simplicity, or something else if you have to follow different naming conventions for cloud resources
- `ROLE_NAME`: should be `roles/secretmanager.secretAccessor` - so you make the pod only be able to access secrets on Secret Manager

#### Using Service Accounts directly

Let's assume you have created a service account correctly and attached a appropriate workload identity. It should roughly look like this:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-secrets
  namespace: es
  annotations:
    iam.gke.io/gcp-service-account: example-team-a@my-project.iam.gserviceaccount.com
```

You can reference this particular ServiceAccount in a `SecretStore` or `ClusterSecretStore`. It's important that you also set the `projectID`, `clusterLocation` and `clusterName`. The Namespace on the `serviceAccountRef` is ignored when using a `SecretStore` resource. This is needed to isolate the namespaces properly.

*When filling `clusterLocation` parameter keep in mind if it is Regional or Zonal cluster.*

```yaml
{% include 'gcpsm-wi-secret-store.yaml' %}
```

*You need to give the Google service account the `roles/iam.serviceAccountTokenCreator` role so it can generate a service account token for you (not necessary in the Pod-based Workload Identity bellow)*

#### Using Pod-based Workload Identity

You can attach a Workload Identity directly to the ESO pod. ESO then has access to all the APIs defined in the attached service account policy. You attach the workload identity by (1) creating a service account with a attached workload identity (described above) and (2) using this particular service account in the pod's `serviceAccountName` field.

For this example we will assume that you installed ESO using helm and that you named the chart installation `external-secrets` and the namespace where it lives `es` like:

```sh
helm install external-secrets external-secrets/external-secrets --namespace es
```

Then most of the resources would have this name, the important one here being the k8s service account attached to the external-secrets operator deployment:

```yaml
# ...
      containers:
      - image: ghcr.io/external-secrets/external-secrets:vVERSION
        name: external-secrets
        ports:
        - containerPort: 8080
          protocol: TCP
      restartPolicy: Always
      schedulerName: default-scheduler
      serviceAccount: external-secrets
      serviceAccountName: external-secrets # <--- here
```

The pod now has the identity. Now you need to configure the `SecretStore`.
You just need to set the `projectID`, all other fields can be omitted.

```yaml
{% include 'gcpsm-pod-wi-secret-store.yaml' %}
```

### GCP Service Account authentication

You can use [GCP Service Account](https://cloud.google.com/iam/docs/service-accounts) to authenticate with GCP. These are static, long-lived credentials. A GCP Service Account is a JSON file that needs to be stored in a `Kind=Secret`. ESO will use that Secret to authenticate with GCP. See here how you [manage GCP Service Accounts](https://cloud.google.com/iam/docs/creating-managing-service-accounts).
After creating a GCP Service account go to `IAM & Admin` web UI, click `ADD ANOTHER ROLE` button, add `Secret Manager Secret Accessor` role to this service account.
The `Secret Manager Secret Accessor` role is required to access secrets.

```yaml
{% include 'gcpsm-credentials-secret.yaml' %}
```


#### Update secret store
Be sure the `gcpsm` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'gcpsm-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `SecretAccessKeyRef` with the namespace of the secret that we just created.

#### Creating external secret

To create a kubernetes secret from the GCP Secret Manager secret a `Kind=ExternalSecret` is needed.

```yaml
{% include 'gcpsm-external-secret.yaml' %}
```

The operator will fetch the GCP Secret Manager secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

### PushSecret owning an existing Google Secret Manager Secret

There are some use cases where you want to use PushSecret for an existing Google Secret Manager Secret that already has labels defined. For example when the creation of the secret is managed by another controller like Kubernetes Config Connector (KCC) and the updating of the secret is managed by ESO.

To allow ESO to take ownership of the existing Google Secret Manager Secret, you need to add the label `"managed-by": "external-secrets"`.

By default, the PushSecret spec will replace any existing labels on the existing GCP Secret Manager Secret. To prevent this, a new field was added to the `spec.data.metadata` object called `mergePolicy` which defaults to `Replace` to ensure that there are no breaking changes and is backward compatible. The other option for this field is `Merge` which will merge the existing labels on the Google Secret Manager Secret with the labels defined in the PushSecret spec. This ensures that the existing labels defined on the Google Secret Manager Secret are retained.

Example of using the `mergePolicy` field:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
  namespace: default
spec:
  updatePolicy: Replace
  deletionPolicy: None
  refreshInterval: 1h
  secretStoreRefs:
    - name: gcp-secretstore
      kind: SecretStore
  selector:
    secret:
      name: bestpokemon
  template:
    data:
      bestpokemon: "{{ .bestpokemon }}"
  data:
    - conversionStrategy: None
      metadata:
        mergePolicy: Merge
        labels:
          anotherLabel: anotherValue
      match:
        secretKey: bestpokemon
        remoteRef:
          remoteKey: best-pokemon
{% endraw %}
```

### Secret Replication and Encryption Configuration

#### Location and Replication

By default, secrets are automatically replicated across multiple regions. You can specify a single location for your secrets by setting the `location` field:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      projectID: my-project
      location: us-east1  # Specify a single location
```

#### Customer-Managed Encryption Keys (CMEK)

You can use your own encryption keys to encrypt secrets at rest. To use Customer-Managed Encryption Keys (CMEK), you need to:

1. Create a Cloud KMS key
2. Grant the service account the `roles/cloudkms.cryptoKeyEncrypterDecrypter` role on the key
3. Specify the key in the PushSecret metadata

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ... other fields ...
  data:
    - match:
        secretKey: mykey
        remoteRef:
          remoteKey: my-secret
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          cmekKeyName: "projects/my-project/locations/us-east1/keyRings/my-keyring/cryptoKeys/my-key"
```

Note: When using CMEK, you must specify a location in the SecretStore as customer-managed encryption keys are region-specific.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      projectID: my-project
      location: us-east1  # Required when using CMEK
```

### Migration Guide: PushSecret Metadata Format (v0.11.x to v0.12.0)

In version 0.12.0, the metadata format for PushSecrets has been standardized to use a structured format. If you're upgrading from v0.11.x, you'll need to update your PushSecret specifications.

#### Old Format (v0.11.x)
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
spec:
  data:
    - match:
        secretKey: mykey
        remoteRef:
          remoteKey: my-secret
      metadata:
        annotations:
          key1: "value1"
        labels:
          key2: "value2"
        topics:
          - "topic1"
          - "topic2"
```

#### New Format (v0.12.0+)
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
spec:
  data:
    - match:
        secretKey: mykey
        remoteRef:
          remoteKey: my-secret
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          annotations:
            key1: "value1"
          labels:
            key2: "value2"
          topics:
            - "topic1"
            - "topic2"
          cmekKeyName: "projects/my-project/locations/us-east1/keyRings/my-keyring/cryptoKeys/my-key"  # Optional: for CMEK
```
