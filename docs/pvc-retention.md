# PVC Retention Policy

## Overview

Nuwa provides **configurable PVC retention policy** to control whether PersistentVolumeClaims are retained or deleted when a Nuwa instance is deleted. This gives you flexibility to choose the right behavior for your use case.

## Retention Policies

### Retain (Default)

PVC is **retained** when Nuwa instance is deleted. This is the default and recommended policy for production environments.

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: my-app
spec:
  image: nginx:latest
  storage:
    type: PVC
    size: 10Gi
    mountPath: /data
    retainPolicy: Retain  # Default, can be omitted
```

**Use Cases:**
- Production databases
- Stateful applications with important data
- Development environments where data needs to persist across deployments
- Any scenario where accidental deletion protection is needed

### Delete

PVC is **automatically deleted** when Nuwa instance is deleted.

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: test-app
spec:
  image: nginx:latest
  storage:
    type: PVC
    size: 1Gi
    mountPath: /data
    retainPolicy: Delete  # PVC will be deleted with Nuwa
```

**Use Cases:**
- Temporary test environments
- CI/CD pipelines
- Ephemeral workloads
- Scenarios where automatic cleanup is desired

## Behavior Comparison

| Policy | Nuwa Deleted | PVC Status | Data Status | Manual Cleanup |
|--------|--------------|------------|-------------|----------------|
| **Retain** | ✅ | ✅ Retained | ✅ Preserved | ⚠️ Required |
| **Delete** | ✅ | ❌ Deleted | ❌ Lost | ✅ Automatic |

## Resource Lifecycle

### Retain Policy

```
Create Nuwa → Create PVC → Create CloneSet → Create Service
                ↓
Delete Nuwa → Delete CloneSet → Delete Service
                ↓
              PVC RETAINED ✅
                ↓
         (Manual cleanup required)
```

### Delete Policy

```
Create Nuwa → Create PVC (with OwnerReference) → Create CloneSet → Create Service
                ↓
Delete Nuwa → Delete CloneSet → Delete Service → Delete PVC
                ↓
              ALL CLEANED UP ✅
```

## Implementation Details

### Owner Reference Control

The retention policy is implemented by conditionally setting the Kubernetes OwnerReference:

```go
// Retain policy: NO owner reference
retainPolicy := nuwa.Spec.Storage.RetainPolicy
if retainPolicy == "" {
    retainPolicy = appv1.PVCRetainPolicyRetain // Default
}

if retainPolicy == appv1.PVCRetainPolicyDelete {
    // Set owner reference - PVC will be cascade deleted
    controllerutil.SetControllerReference(nuwa, pvc, r.Scheme)
}
// If Retain: no owner reference - PVC is independent
```

### Tracking Annotations

PVC includes annotations to track its configuration:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-app-pvc
  annotations:
    nuwa.12306.work/created-by: "my-app"
    nuwa.12306.work/retain-policy: "Retain"  # or "Delete"
  labels:
    app: my-app
```

## Usage Examples

### 1. Production App with Retain Policy (Default)

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: prod-database
spec:
  image: postgres:15
  replicas: 1
  storage:
    type: PVC
    size: 50Gi
    storageClassName: fast-ssd
    mountPath: /var/lib/postgresql/data
    retainPolicy: Retain  # Explicit, but this is the default
```

**Behavior:**
- Delete Nuwa: `kubectl delete nuwa prod-database`
- PVC `prod-database-pvc` is retained
- Data is safe and can be reattached

### 2. Test Environment with Delete Policy

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: test-app
spec:
  image: nginx:latest
  replicas: 1
  storage:
    type: PVC
    size: 1Gi
    mountPath: /data
    retainPolicy: Delete  # Auto-cleanup
```

**Behavior:**
- Delete Nuwa: `kubectl delete nuwa test-app`
- PVC `test-app-pvc` is automatically deleted
- No manual cleanup needed

### 3. Dynamic Policy Changes

The controller supports **safe direction** policy changes dynamically:

| Change Direction | Supported | Reason |
|-----------------|-----------|--------|
| Delete → Retain | ✅ Yes | Safe: preserves data |
| Retain → Delete | ❌ No | Dangerous: could cause data loss |

**Delete → Retain (Supported):**
```bash
# Change from Delete to Retain - takes effect immediately
kubectl patch nuwa my-app --type='json' -p='[
  {"op": "replace", "path": "/spec/storage/retainPolicy", "value": "Retain"}
]'

# The controller will automatically remove the owner reference
# PVC will now be preserved when Nuwa is deleted
```

**Retain → Delete (Not Supported Dynamically):**
```bash
# This change requires recreation to prevent accidental data loss
# 1. First, backup your data if needed

# 2. Delete the Nuwa instance
kubectl delete nuwa my-app

# 3. Manually delete the retained PVC
kubectl delete pvc my-app-pvc

# 4. Recreate with Delete policy
kubectl apply -f my-app.yaml  # with retainPolicy: Delete
```

**Why Retain → Delete is not supported dynamically:**
- Prevents accidental data loss from simple configuration changes
- Requires explicit user action (delete + recreate) as confirmation
- Follows the principle of least surprise

## Comparison with Other Resources

| Resource | Deletion Behavior | Reason |
|----------|------------------|---------|
| **PVC** | ✅ Retained | Data persistence |
| **CloneSet** | ❌ Deleted | Stateless workload |
| **Service** | ❌ Deleted | Network configuration |

## Best Practices

### 1. Choose the Right Policy

| Scenario | Recommended Policy | Reason |
|----------|-------------------|---------|
| Production databases | **Retain** | Data safety is critical |
| Stateful applications | **Retain** | Preserve application state |
| Development/staging | **Retain** | Reuse data across deployments |
| CI/CD test environments | **Delete** | Automatic cleanup |
| Temporary workloads | **Delete** | No manual cleanup needed |
| Ephemeral caches | **Delete** | Data is disposable |

### 2. Naming Convention

PVC name follows the pattern: `{nuwa-name}-pvc`

```bash
# Nuwa: my-app
# PVC:  my-app-pvc
```

### 3. Manual Cleanup for Retained PVCs

**Find Retained PVCs:**
```bash
# List all PVCs with Retain policy
kubectl get pvc -A -o json | \
  jq -r '.items[] | select(.metadata.annotations["nuwa.12306.work/retain-policy"] == "Retain") |
  "\(.metadata.namespace)/\(.metadata.name) (created by: \(.metadata.annotations["nuwa.12306.work/created-by"]))"'
```

**Find Orphaned PVCs:**
```bash
# Find PVCs where the Nuwa instance no longer exists
for pvc in $(kubectl get pvc -o name); do
  nuwa_name=$(kubectl get $pvc -o jsonpath='{.metadata.annotations.nuwa\.12306\.work/created-by}')
  if [ -n "$nuwa_name" ] && ! kubectl get nuwa $nuwa_name &>/dev/null; then
    echo "Orphaned: $pvc (Nuwa: $nuwa_name)"
  fi
done
```

**Cleanup Script:**
```bash
#!/bin/bash
# cleanup-orphaned-pvcs.sh

for pvc in $(kubectl get pvc -o name); do
  nuwa_name=$(kubectl get $pvc -o jsonpath='{.metadata.annotations.nuwa\.12306\.work/created-by}')
  policy=$(kubectl get $pvc -o jsonpath='{.metadata.annotations.nuwa\.12306\.work/retain-policy}')

  if [ "$policy" = "Retain" ] && [ -n "$nuwa_name" ]; then
    if ! kubectl get nuwa $nuwa_name &>/dev/null; then
      echo "Orphaned PVC: $pvc (Nuwa: $nuwa_name)"
      # Uncomment to delete:
      # kubectl delete $pvc
    fi
  fi
done
```

### 4. Data Recovery

**Retain Policy - Reattach Data:**
```bash
# 1. Delete Nuwa (PVC is retained)
kubectl delete nuwa my-app

# 2. Recreate Nuwa with same name
kubectl apply -f my-app.yaml

# 3. Data is automatically reattached
```

**Delete Policy - No Recovery:**
```bash
# Once Nuwa is deleted, PVC and data are gone
# Make sure to backup data before deletion if needed
```

## Troubleshooting

### PVC Stuck in Terminating

If PVC is stuck when trying to delete:

```bash
# Check if PVC is still mounted
kubectl get pods -o json | \
  jq -r '.items[] | select(.spec.volumes[]?.persistentVolumeClaim.claimName == "my-app-pvc") | .metadata.name'

# Delete pods first, then PVC
kubectl delete pod <pod-name>
kubectl delete pvc my-app-pvc
```

### PVC Not Reattached

If recreating Nuwa doesn't reattach PVC:

```bash
# Verify PVC name matches
kubectl get pvc my-app-pvc

# Check PVC status
kubectl describe pvc my-app-pvc

# Verify Nuwa name matches annotation
kubectl get pvc my-app-pvc -o jsonpath='{.metadata.annotations.nuwa\.12306\.work/created-by}'
```

## Migration Guide

### From Previous Version (with Owner Reference)

If upgrading from a version where PVC had owner reference:

1. **Existing PVCs**: Will continue to work but will be deleted with Nuwa
2. **New PVCs**: Will be retained when Nuwa is deleted
3. **To Migrate**: Remove owner reference from existing PVCs:

```bash
# Remove owner reference from existing PVC
kubectl patch pvc my-app-pvc --type=json -p='[{"op": "remove", "path": "/metadata/ownerReferences"}]'

# Add retention annotation
kubectl annotate pvc my-app-pvc nuwa.12306.work/retain=true
```

## Future Enhancements

Potential future features:

- [ ] Configurable retention policy via Nuwa spec
- [ ] Automatic PVC cleanup after retention period
- [ ] PVC snapshot before deletion
- [ ] PVC resize support
