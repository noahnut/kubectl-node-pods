package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

type options struct {
	kubeConfig string
	kubeCtx    string
	namespace  string
}

type nodeStats struct {
	name            string
	status          string
	roles           string
	version         string
	podCount        int
	allocCPUm       int64
	reqCPUm         int64
	limitCPUm       int64
	allocMemMi      int64
	reqMemMi        int64
	limitMemMi      int64
	numVolumes      int
	requestPressure string
}

func main() {
	opts := options{}

	cmd := &cobra.Command{
		Use:   "kubectl-node_pods",
		Short: "Show node pod and resource balance",
		Long:  "A kubectl plugin that displays pod counts and CPU/memory request pressure on each node.",
		Example: `  # Show pod and resource balance for all namespaces
  kubectl node-pods

  # Include kube-system only
  kubectl node-pods -n kube-system

  # Use a specific kubeconfig
  kubectl node-pods --kubeconfig /path/to/config

  # Use a specific context
  kubectl node-pods --context my-cluster`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}

	cmd.Flags().StringVar(&opts.kubeConfig, "kubeconfig", "", "path to the kubeconfig file")
	cmd.Flags().StringVar(&opts.kubeCtx, "context", "", "the kubeconfig context to use")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "filter pods by namespace (default: all namespaces)")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(opts options) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.kubeConfig != "" {
		loadingRules.ExplicitPath = opts.kubeConfig
	}

	overrides := &clientcmd.ConfigOverrides{}
	if opts.kubeCtx != "" {
		overrides.CurrentContext = opts.kubeCtx
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return fmt.Errorf("build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	if len(nodes.Items) == 0 {
		return fmt.Errorf("no nodes found")
	}

	ns := opts.namespace
	if ns == "" {
		ns = corev1.NamespaceAll
	}
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pods: %w", err)
	}

	statsByNode := make(map[string]*nodeStats, len(nodes.Items))
	for _, n := range nodes.Items {
		statsByNode[n.Name] = &nodeStats{
			name:       n.Name,
			status:     nodeReadyStatus(n),
			roles:      nodeRoles(n),
			version:    n.Status.NodeInfo.KubeletVersion,
			allocCPUm:  n.Status.Allocatable.Cpu().MilliValue(),
			allocMemMi: n.Status.Allocatable.Memory().Value() / (1024 * 1024),
			numVolumes: len(n.Status.VolumesAttached),
		}
	}

	for _, p := range pods.Items {
		if p.Spec.NodeName == "" {
			continue
		}
		ns, ok := statsByNode[p.Spec.NodeName]
		if !ok {
			continue
		}
		ns.podCount++
		reqCPU, limitCPU, reqMem, limitMem := podResourceTotals(&p)
		ns.reqCPUm += reqCPU
		ns.limitCPUm += limitCPU
		ns.reqMemMi += reqMem
		ns.limitMemMi += limitMem
	}

	var stats []nodeStats
	for _, s := range statsByNode {
		s.requestPressure = pressureLabel(percent(s.reqCPUm, s.allocCPUm), percent(s.reqMemMi, s.allocMemMi))
		stats = append(stats, *s)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].podCount > stats[j].podCount
	})

	printStats(stats, opts.namespace)
	return nil
}

func printStats(stats []nodeStats, namespace string) {
	tw := tabwriter.NewWriter(os.Stdout, 2, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NODE\tSTATUS\tROLES\tVERSION\tPODS\tCPU_REQ/ALLOC\tMEM_REQ/ALLOC\tVOLUMES\tPRESSURE")

	var totalPods int
	var totalReqCPUm, totalAllocCPUm int64
	var totalReqMemMi, totalAllocMemMi int64
	for _, s := range stats {
		totalPods += s.podCount
		totalReqCPUm += s.reqCPUm
		totalAllocCPUm += s.allocCPUm
		totalReqMemMi += s.reqMemMi
		totalAllocMemMi += s.allocMemMi
		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\t%d\t%s\n",
			s.name,
			s.status,
			s.roles,
			s.version,
			s.podCount,
			fmt.Sprintf("%dm/%dm (%.1f%%)", s.reqCPUm, s.allocCPUm, percent(s.reqCPUm, s.allocCPUm)),
			fmt.Sprintf("%dMi/%dMi (%.1f%%)", s.reqMemMi, s.allocMemMi, percent(s.reqMemMi, s.allocMemMi)),
			s.numVolumes,
			s.requestPressure,
		)
	}

	fmt.Fprintln(tw)
	scope := "all namespaces"
	if namespace != "" {
		scope = fmt.Sprintf("namespace %q", namespace)
	}
	fmt.Fprintf(
		tw,
		"TOTAL\t-\t-\t-\t%d\t%dm/%dm (%.1f%%)\t%dMi/%dMi (%.1f%%)\t%s\n",
		totalPods,
		totalReqCPUm, totalAllocCPUm, percent(totalReqCPUm, totalAllocCPUm),
		totalReqMemMi, totalAllocMemMi, percent(totalReqMemMi, totalAllocMemMi),
		scope,
	)
	tw.Flush()
}

func podResourceTotals(pod *corev1.Pod) (reqCPUm, limitCPUm, reqMemMi, limitMemMi int64) {
	var appReqCPU, appLimitCPU, appReqMem, appLimitMem int64
	for _, c := range pod.Spec.Containers {
		appReqCPU += c.Resources.Requests.Cpu().MilliValue()
		appLimitCPU += c.Resources.Limits.Cpu().MilliValue()
		appReqMem += c.Resources.Requests.Memory().Value() / (1024 * 1024)
		appLimitMem += c.Resources.Limits.Memory().Value() / (1024 * 1024)
	}

	var maxInitReqCPU, maxInitLimitCPU, maxInitReqMem, maxInitLimitMem int64
	for _, c := range pod.Spec.InitContainers {
		maxInitReqCPU = max(maxInitReqCPU, c.Resources.Requests.Cpu().MilliValue())
		maxInitLimitCPU = max(maxInitLimitCPU, c.Resources.Limits.Cpu().MilliValue())
		maxInitReqMem = max(maxInitReqMem, c.Resources.Requests.Memory().Value()/(1024*1024))
		maxInitLimitMem = max(maxInitLimitMem, c.Resources.Limits.Memory().Value()/(1024*1024))
	}

	reqCPUm = max(appReqCPU, maxInitReqCPU)
	limitCPUm = max(appLimitCPU, maxInitLimitCPU)
	reqMemMi = max(appReqMem, maxInitReqMem)
	limitMemMi = max(appLimitMem, maxInitLimitMem)

	if pod.Spec.Overhead != nil {
		reqCPUm += pod.Spec.Overhead.Cpu().MilliValue()
		reqMemMi += pod.Spec.Overhead.Memory().Value() / (1024 * 1024)
	}
	return reqCPUm, limitCPUm, reqMemMi, limitMemMi
}

func nodeReadyStatus(node corev1.Node) string {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodeRoles(node corev1.Node) string {
	var roles []string
	for k := range node.Labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			if role == "" {
				role = "worker"
			}
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return "worker"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

func pressureLabel(cpuPct, memPct float64) string {
	maxPct := cpuPct
	if memPct > maxPct {
		maxPct = memPct
	}
	switch {
	case maxPct >= 90:
		return "high"
	case maxPct >= 70:
		return "medium"
	default:
		return "low"
	}
}

func percent(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(used) / float64(total)) * 100
}
