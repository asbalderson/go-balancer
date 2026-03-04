package discovery

import (
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"pkg/logging"
)

func createClient(kubeconfigPath string) (kubernetes.Interface, error) {
	var kubeconf *rest.Config

	if kubeconfigPath != "" {
		conf, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconf from %s: %v", kubeconfigPath, err)
		}
		kubeconf = conf
	} else {
		conf, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeconf = conf
	}

	client, err := kubernetes.NewForConfig(kubeconf)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %v", err)
	}

	logging.Debug("Created a Kubernetes client")
	return client, nil
}

type Backend struct {
	Address string
	PodName string
}

type BackendList struct {
	mu       sync.RWMutex
	backends []Backend
}

func NewBackendList() *BackendList {
	return &BackendList{}
}

func (bl *BackendList) Replace(backends []Backend) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.backends = backends
}

func (bl *BackendList) GetAll() []Backend {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	result := make([]Backend, len(bl.backends))
	copy(result, bl.backends)
	return result
}

func reconcile(endpoints *corev1.Endpoints, backendList *BackendList, serviceName string) {
	name := endpoints.Name
	if name != serviceName {
		return
	}
	logging.Debug("Detected an update for service %s, updating now", serviceName)
	var backends []Backend
	for _, subnet := range endpoints.Subsets {
		for _, address := range subnet.Addresses {
			ip := address.IP
			podName := address.TargetRef.Name
			logging.Debug("Adding pod %s", podName)
			backends = append(backends, Backend{
				Address: ip,
				PodName: podName,
			})
		}
	}
	backendList.Replace(backends)
	logging.Debug("Service pods updated")
}

func GetBackendFactory(kubeconfPath string) (informers.SharedInformerFactory, error) {
	client, err := createClient(kubeconfPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load kubeconf: %w", err)
	}

	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		return nil, fmt.Errorf("Namespace not set in environment")
	}

	factory := informers.NewSharedInformerFactoryWithOptions(client, 3*time.Minute, informers.WithNamespace(namespace))
	logging.Debug("Informer created, will watch for service pods to populate")
	return factory, nil
}

func GetBackends(factory informers.SharedInformerFactory, serviceName string) *BackendList {
	endpointInformer := factory.Core().V1().Endpoints().Informer()

	backendList := NewBackendList()

	endpointInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints, ok := obj.(*corev1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendList, serviceName)
		},
		UpdateFunc: func(old, obj interface{}) {
			endpoints, ok := obj.(*corev1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendList, serviceName)
		},
		DeleteFunc: func(obj interface{}) {
			endpoints, ok := obj.(*corev1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendList, serviceName)
		},
	})

	return backendList
}
