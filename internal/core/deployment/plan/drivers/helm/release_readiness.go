package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

type WorkloadRef struct {
	Kind      string
	Namespace string
	Name      string
}

// ExtractWorkloadsFromManifest parses Helm release manifest and returns workload refs.
// It is tolerant to empty documents and comments.
func ExtractWorkloadsFromManifest(manifest string, defaultNamespace string) ([]WorkloadRef, error) {
	manifest = strings.TrimSpace(manifest)
	if manifest == "" {
		return nil, nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(manifest)), 4096)

	type key struct{ kind, ns, name string }
	seen := map[key]struct{}{}
	var out []WorkloadRef

	for {
		var m map[string]any
		err := decoder.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode manifest: %w", err)
		}
		if len(m) == 0 {
			continue
		}

		kind, _ := m["kind"].(string)
		if !isSupportedWorkloadKind(kind) {
			continue
		}

		metaMap, _ := m["metadata"].(map[string]any)
		name, _ := metaMap["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		namespace, _ := metaMap["namespace"].(string)
		if strings.TrimSpace(namespace) == "" {
			namespace = defaultNamespace
		}
		k := key{kind: kind, ns: namespace, name: name}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, WorkloadRef{Kind: kind, Namespace: namespace, Name: name})
	}

	// Stable order for deterministic messages.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func isSupportedWorkloadKind(kind string) bool {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		return true
	default:
		return false
	}
}

type WorkloadCheckResult struct {
	Ready  bool
	Failed bool
	Reason string // short human message
}

// CheckReleaseWorkloadsReady checks whether workloads referenced by manifest are truly ready in k8s.
// Returns:
// - allReady: all workloads are ready (or no workloads)
// - anyFailed: at least one workload is in a terminal failed state
// - message: aggregated status / reasons for non-ready or failed workloads
func CheckReleaseWorkloadsReady(ctx context.Context, restClientGetter *RESTClientGetter, manifest string, defaultNamespace string) (allReady bool, anyFailed bool, message string, err error) {
	refs, err := ExtractWorkloadsFromManifest(manifest, defaultNamespace)
	if err != nil {
		return false, false, "", err
	}
	// If chart has no workloads, treat as ready.
	if len(refs) == 0 {
		return true, false, "", nil
	}

	restCfg, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return false, false, "", err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return false, false, "", err
	}

	var notReady []string
	var failed []string

	for _, ref := range refs {
		res := checkOneWorkload(ctx, clientset, ref)
		if res.Failed {
			anyFailed = true
			failed = append(failed, fmt.Sprintf("%s/%s: %s", ref.Kind, ref.Name, res.Reason))
			continue
		}
		if !res.Ready {
			notReady = append(notReady, fmt.Sprintf("%s/%s: %s", ref.Kind, ref.Name, res.Reason))
		}
	}

	if anyFailed {
		return false, true, summarizeReasons("failed", failed, 6), nil
	}
	if len(notReady) > 0 {
		return false, false, summarizeReasons("running", notReady, 6), nil
	}
	return true, false, "", nil
}

func summarizeReasons(prefix string, items []string, limit int) string {
	if len(items) == 0 {
		return prefix
	}
	if limit <= 0 || len(items) <= limit {
		return fmt.Sprintf("%s: %s", prefix, strings.Join(items, "; "))
	}
	head := items[:limit]
	return fmt.Sprintf("%s: %s; ... and %d more", prefix, strings.Join(head, "; "), len(items)-limit)
}

func checkOneWorkload(ctx context.Context, clientset *kubernetes.Clientset, ref WorkloadRef) WorkloadCheckResult {
	switch ref.Kind {
	case "Deployment":
		obj, err := clientset.AppsV1().Deployments(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: "not found"}
		}
		if err != nil {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: err.Error()}
		}
		return checkDeployment(obj)
	case "StatefulSet":
		obj, err := clientset.AppsV1().StatefulSets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: "not found"}
		}
		if err != nil {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: err.Error()}
		}
		return checkStatefulSet(obj)
	case "DaemonSet":
		obj, err := clientset.AppsV1().DaemonSets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: "not found"}
		}
		if err != nil {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: err.Error()}
		}
		return checkDaemonSet(obj)
	case "Job":
		obj, err := clientset.BatchV1().Jobs(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: "not found"}
		}
		if err != nil {
			return WorkloadCheckResult{Ready: false, Failed: false, Reason: err.Error()}
		}
		return checkJob(obj)
	default:
		return WorkloadCheckResult{Ready: true, Failed: false, Reason: "ignored"}
	}
}

func checkDeployment(d *appsv1.Deployment) WorkloadCheckResult {
	var replicas int32 = 1
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	}

	observedOK := d.Status.ObservedGeneration >= d.Generation
	ready := observedOK &&
		d.Status.UpdatedReplicas >= replicas &&
		d.Status.AvailableReplicas >= replicas

	if ready {
		return WorkloadCheckResult{Ready: true}
	}

	// Hard fail: progress deadline exceeded
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == "False" && c.Reason == "ProgressDeadlineExceeded" {
			return WorkloadCheckResult{Ready: false, Failed: true, Reason: "progress deadline exceeded"}
		}
	}

	return WorkloadCheckResult{
		Ready:  false,
		Failed: false,
		Reason: fmt.Sprintf("available %d/%d (updated %d/%d)", d.Status.AvailableReplicas, replicas, d.Status.UpdatedReplicas, replicas),
	}
}

func checkStatefulSet(s *appsv1.StatefulSet) WorkloadCheckResult {
	var replicas int32 = 1
	if s.Spec.Replicas != nil {
		replicas = *s.Spec.Replicas
	}
	observedOK := s.Status.ObservedGeneration >= s.Generation
	ready := observedOK && s.Status.ReadyReplicas >= replicas
	if ready {
		return WorkloadCheckResult{Ready: true}
	}
	return WorkloadCheckResult{
		Ready:  false,
		Failed: false,
		Reason: fmt.Sprintf("ready %d/%d", s.Status.ReadyReplicas, replicas),
	}
}

func checkDaemonSet(d *appsv1.DaemonSet) WorkloadCheckResult {
	observedOK := d.Status.ObservedGeneration >= d.Generation
	desired := d.Status.DesiredNumberScheduled
	ready := observedOK && d.Status.NumberReady >= desired && d.Status.UpdatedNumberScheduled >= desired
	if ready {
		return WorkloadCheckResult{Ready: true}
	}
	return WorkloadCheckResult{
		Ready:  false,
		Failed: false,
		Reason: fmt.Sprintf("ready %d/%d (updated %d/%d)", d.Status.NumberReady, desired, d.Status.UpdatedNumberScheduled, desired),
	}
}

func checkJob(j *batchv1.Job) WorkloadCheckResult {
	// Terminal failed if condition Failed is True, or failed >= backoffLimit (if set/defaulted).
	for _, c := range j.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == "True" {
			r := strings.TrimSpace(c.Reason)
			if r == "" {
				r = "job failed"
			}
			return WorkloadCheckResult{Ready: false, Failed: true, Reason: r}
		}
	}

	var completions int32 = 1
	if j.Spec.Completions != nil {
		completions = *j.Spec.Completions
	}
	if j.Status.Succeeded >= completions {
		return WorkloadCheckResult{Ready: true}
	}

	var backoffLimit int32 = 6
	if j.Spec.BackoffLimit != nil {
		backoffLimit = *j.Spec.BackoffLimit
	}
	// Controller marks job failed when failed > backoffLimit.
	if j.Status.Failed > backoffLimit {
		return WorkloadCheckResult{Ready: false, Failed: true, Reason: fmt.Sprintf("failed %d > backoffLimit %d", j.Status.Failed, backoffLimit)}
	}

	return WorkloadCheckResult{
		Ready:  false,
		Failed: false,
		Reason: fmt.Sprintf("succeeded %d/%d (failed %d)", j.Status.Succeeded, completions, j.Status.Failed),
	}
}
