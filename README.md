# Migrate

## How to run

```shell
go run migrate --kubeconfig ~/.kube/config    
```

## How to build

```
go build  migrate

docker run --rm -v $(pwd):/app -w /app golang:latest \
   bash -c 'GOOS=linux GOARCH=amd64 go build -o migrate_linux_amd64'
```

## ARM NodePool Example

```yaml
template:
  metadata:
    labels:
      node.cloudpilot.ai/managed: "true"
  spec:
    taints:
      - key: node.cloudpilot.ai/arch-arm64
        effect: NoSchedule
    requirements:
      - key: karpenter.k8s.aws/instance-category
        operator: NotIn
        values:
          - p
          - g
          - gr
          - inf
          - a
      - key: kubernetes.io/arch
        operator: In
        values:
          - arm64
      - key: kubernetes.io/os
        operator: In
        values:
          - linux
      - key: karpenter.sh/capacity-type
        operator: In
        values:
          - spot
          - on-demand
      - key: karpenter.k8s.aws/instance-memory
        operator: Lt
        values:
          - "32769"
      - key: karpenter.k8s.aws/instance-cpu
        operator: Lt
        values:
          - "17"
      - key: beta.kubernetes.io/instance-type
        operator: NotIn
        values:
          - c1.medium
          - m1.small
    nodeClassRef:
      kind: EC2NodeClass
      name: cloudpilot
      group: karpenter.k8s.aws
    expireAfter: Never
disruption:
  consolidateAfter: 60m
  consolidationPolicy: WhenEmptyOrUnderutilized
  budgets:
    - nodes: "2"
weight: 2
```
