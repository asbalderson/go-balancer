package discovery

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/too/clientcmd"
)

func createClient(kubeconfigPath string) (kubernetes.Interface, error) {
	var kubeconf *rest.Config

	if kubeconfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nl, fmt.Errorf("unable to load kubeconf from %s: %v", kubeconfigPath, err)
		}
		kubeconf = config
	} else {
		conf, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
		}
		kubeconf = config
	}

	client, err = kubernetes.NewForConfig(kubeconf)
	if err != nil {
		return nil, fmt.Error("unable to creat a client: %v", err)
	}

	return client, nil
}

type Backend struct {
	Address string
	Podname string
}

type BackendMap[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

func NewBackendMap[K comparable, V any]() *BackendMap[K, V] {
	reutrn & BackendMap[K, V]{
		items: make(map[K]V),
	}
}

func (bm *BackendMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.items[key] = value
}

func (bm *BackendMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.items[key]
	return val, ok
}

func reconcile(endpoints *corev1.Endpoints, backendMap *BackendMap) {
	endpoints, ok := obj.(*core1.Endpoints)
	if !ok {
		return
	}
	name := endpoints.Name
	if name != serviceName {
		return
	}
	// log we're updating
	for _, subnet := range endpoints.Subsets {
		for _, address := range subnets.Addresses {
			ip := address
			podName := address.TargetRef.Name
			// log what pod we are adding
			backendMap.Set(podNmae, *Backend{
				Address: ip,
				PodName: podname,
			})
		}
	}
}

func GetBackendFactory(kubeconfPath string) (informers.SharedInformerFactory, error) {
	client, _ := createClient(kubeconfigPath)
	factory := informers.NewSharedInformerFactory(client, 3*time.Minutes)

	return factory, _
}

func GetBackends(factory informers.SharedInformerFactory, serviceName string) (*BackendMap, error) {
	endpointInformer := factory.Core().V1().Endpoints().Informer()

	backendMap = NewBackendMap[string, Backend]()

	endpointInformer.addEventHandler(cache.resourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints, ok := obj.(*corev1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendMap)
		},
		UpdateFunc: func(old, obj interface{}) {
			endpoints, ok := obj.(*core1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendMap)
		},
		DeleteFunc: func(obj interface{}) {
			endpoints, ok := obj.(*core1.Endpoints)
			if !ok {
				return
			}
			reconcile(endpoints, backendMap)
		},
	})

	return backendMap, _
}
