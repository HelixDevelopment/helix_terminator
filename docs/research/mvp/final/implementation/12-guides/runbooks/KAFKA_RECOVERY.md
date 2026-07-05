# KAFKA_RECOVERY.md

## 1. Objective

Recover from Kafka broker failures, partition under-replication, and data unavailability in the helix_terminator event streaming backbone.

## 2. Prerequisites

- Kafka cluster is managed by Strimzi or self-hosted on Kubernetes.
- `kafka-topics.sh`, `kafka-broker-api-versions.sh`, and `kubectl` are available.
- ZooKeeper (or KRaft) is accessible.
- Monitoring: Grafana dashboards for `KafkaUnderReplicatedPartitions` and `KafkaOfflinePartitions`.

## 3. Scenario A: Single Broker Failure

### Symptoms
- Alert: `KafkaUnderReplicatedPartitions > 0`
- One broker pod is in `CrashLoopBackOff` or `NotReady`.

### Step-by-Step Recovery

```bash
# Step 1: Identify the failed broker
kubectl get pods -n data -l app=kafka
# Example: kafka-broker-2   0/1   CrashLoopBackOff

# Step 2: Inspect broker logs
kubectl logs kafka-broker-2 -n data --tail=200 | grep -i "error\|fatal\|exception"

# Step 3: Check disk space (common cause)
kubectl exec -it kafka-broker-2 -n data -- df -h /var/lib/kafka

# Step 4: If disk is full, expand the PVC
kubectl patch pvc data-kafka-broker-2 -n data --type=merge -p \
  '{"spec":{"resources":{"requests":{"storage":"500Gi"}}}}'

# Step 5: If the broker is stuck, delete the pod (StatefulSet will recreate it)
kubectl delete pod kafka-broker-2 -n data

# Step 6: Wait for the pod to become Ready
kubectl rollout status statefulset/kafka-broker -n data

# Step 7: Verify under-replicated partitions return to zero
kubectl exec -it kafka-broker-0 -n data -- \
  kafka-topics.sh --bootstrap-server localhost:9092 --describe | grep -i "under-replicated"
```

## 4. Scenario B: Multiple Broker Failure (Loss of Quorum)

### Symptoms
- `KafkaOfflinePartitions > 0`
- More than `min.insync.replicas` brokers are down.
- Producers may be failing with `NOT_ENOUGH_REPLICAS`.

### Step-by-Step Recovery

```bash
# Step 1: Identify all failed brokers
kubectl get pods -n data -l app=kafka

# Step 2: If the failure is due to a zone outage, verify node status
kubectl get nodes -l topology.kubernetes.io/zone=us-east-1a

# Step 3: If brokers are unrecoverable, force leader election for critical topics
# (Only if you accept potential data loss for unclean leader election)

# WARNING: Unclean leader election can cause data loss. Use only in emergencies.
kubectl exec -it kafka-broker-0 -n data -- \
  kafka-leader-election.sh --bootstrap-server localhost:9092 \
  --election-type preferred --topic helix-events --partition 0

# Step 4: If the cluster is managed by Strimzi, check the Kafka CR status
kubectl get kafka helix-kafka -n data -o jsonpath='{.status.conditions}' | jq

# Step 5: If a full cluster rebuild is needed, restore from backup
# (Assuming S3 backup via Kafka Connect or MirrorMaker 2)
# See the DR runbook for full cluster restore.
```

## 5. Scenario C: Corrupted Log Segment

```bash
# Step 1: Identify the corrupted partition from broker logs
kubectl logs kafka-broker-1 -n data | grep -i "corrupt"

# Step 2: Move the corrupted segment aside (on the broker)
kubectl exec -it kafka-broker-1 -n data -- bash -c \
  "mv /var/lib/kafka/data/helix-events-0/00000000000000012345.log /tmp/corrupted/"

# Step 3: Restart the broker
kubectl delete pod kafka-broker-1 -n data

# Step 4: Verify partition recovery
kubectl exec -it kafka-broker-0 -n data -- \
  kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic helix-events
```

## 6. Scenario D: Consumer Lag Recovery

```bash
# Step 1: Identify high consumer lag
kubectl exec -it kafka-broker-0 -n data -- \
  kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group audit-service-consumer

# Step 2: If the consumer is stuck, reset offset to latest (emergency only)
# WARNING: This skips unprocessed messages.
kubectl exec -it kafka-broker-0 -n data -- \
  kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group audit-service-consumer --topic helix-events --reset-offsets --to-latest --execute

# Step 3: Scale up consumers to increase parallelism
kubectl scale deployment audit-service -n production --replicas=6
```

## 7. Verification Checklist

- [ ] All broker pods are `Running` and `Ready`.
- [ ] `KafkaUnderReplicatedPartitions` == 0.
- [ ] `KafkaOfflinePartitions` == 0.
- [ ] Producer and consumer throughput metrics are nominal.
- [ ] No `ERROR` or `FATAL` logs in broker stdout.
- [ ] Consumer lag is within acceptable thresholds (< 1000 messages per partition).

## 8. References
- `docs/guides/runbooks/FAILOVER_PROCEDURE.md`
- `infrastructure/helm/kafka/` — Kafka Helm charts
- `docs/research/mvp/final/implementation/12-guides/ADRs/ADR-003-kafka-over-nats.md`
