package kubeclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"k8s.io/client-go/rest"

	sandboxv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/client/generated/clientset/versioned/typed/core/v1alpha1"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
)

type sandbox struct {
	config     *rest.Config
	resource   string
	namespace  string
	restClient *rest.RESTClient
	sandboxv1alpha1.SandboxInterface
}

type connectionStruct struct {
	con StreamInterface
	err error
}

func (s sandbox) Attach(name string, options *subv1alpha1.Attach) (StreamInterface, error) {
	if options == nil || options.ConnectionTimeout.Duration == 0 {
		return asyncSubresourceHelper(s.config, s.resource, s.namespace, name, "attach", url.Values{})
	}

	ticker := time.NewTicker(options.ConnectionTimeout.Duration)
	connectionChan := make(chan connectionStruct)

	go func() {
		for {
			select {
			case <-ticker.C:
				connectionChan <- connectionStruct{
					con: nil,
					err: fmt.Errorf("timeout trying to connect to the virtual machine instance"),
				}
				return
			default:
			}

			con, err := asyncSubresourceHelper(s.config, s.resource, s.namespace, name, "attach", url.Values{})
			if err != nil {
				var asyncSubresourceError *AsyncSubresourceError
				ok := errors.As(err, &asyncSubresourceError)
				// return if response status code does not equal to 400
				if !ok || asyncSubresourceError.GetStatusCode() != http.StatusBadRequest {
					connectionChan <- connectionStruct{con: nil, err: err}
					return
				}

				time.Sleep(1 * time.Second)
				continue
			}

			connectionChan <- connectionStruct{con: con, err: nil}
			return
		}
	}()
	conStruct := <-connectionChan
	return conStruct.con, conStruct.err
}
