package watch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/coreos/etcd/clientv3"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"sync"
	"time"
)

type Watch struct {
	revision int64
	cond     chan struct{}
	rwl      sync.RWMutex
}

//Wait until revision is greater than lastRevision
func (w *Watch) WaitNext(ctx context.Context, lastRevision int64, notify chan<- int64) {
	for {
		w.rwl.RLock()
		if w.revision > lastRevision {
			w.rwl.RUnlock()
			break
		}
		cond := w.cond
		w.rwl.RUnlock()
		select {
		case <-cond:
		case <-ctx.Done():
			return
		}
	}

	//We accept larger revision ,so do not need to use Rlock
	select {
	case notify <- w.revision:
	case <-ctx.Done():
	}
}

//Update watch revision
func (w *Watch) update(newRevision int64) {
	w.rwl.Lock()
	defer w.rwl.Unlock()
	w.revision = newRevision
	close(w.cond)
	w.cond = make(chan struct{})
}

//
func createWatch(cli *clientv3.Client, prefix string) (*Watch, error) {
	w := &Watch{0, make(chan struct{}), sync.RWMutex{}}

	go func() {
		wc := cli.Watch(context.Background(), prefix, clientv3.WithPrefix(), clientv3.WithCreatedNotify())
		log.Debug("Watch created on %s", prefix)

		for {
			for wresp := range wc {
				if wresp.CompactRevision > w.revision {
					// respect CompactRevision
					w.update(wresp.CompactRevision)
					log.Debug("Watch to '%s' updated to '%d' by CompactRevision", prefix, wresp.CompactRevision)
				} else if wresp.Header.GetRevision() > w.revision {
					// Watch created or updated
					w.update(wresp.Header.GetRevision())
					log.Debug("Watch to '%s' updated to %d by header revision", prefix, wresp.Header.GetRevision())
				}

				if err := wresp.Err(); err != nil {
					log.Error("Watch to '%s' err '%v' at revision %d ", prefix, w.revision)
				}
			}
			log.Warn("Watch to '%s' stopped at revision %d ", prefix, w.revision)

			//disconnected or canceled
			//wait for a moment to avoid reconnecting
			//too quickly
			time.Sleep(3 * time.Second)
			if w.revision > 0 {
				//start from next revision so we are not missing anything
				wc = cli.Watch(context.Background(), prefix, clientv3.WithPrefix(), clientv3.WithRev(w.revision+1))
			} else {
				//start from the lastest revision
				wc = cli.Watch(context.Background(), prefix, clientv3.WithPrefix(), clientv3.WithCreatedNotify())
			}
		}
	}()

	return w, nil
}

type Client struct {
	client *clientv3.Client
	watchs map[string]*Watch
	wm     sync.Mutex
}

func NewEtcdClient(machines []string, cert, key, caCert string, basicAuth bool, username, password string) (*Client, error) {
	cfg := clientv3.Config{
		Endpoints:            machines,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 3 * time.Second,
	}

	if basicAuth {
		cfg.Username = username
		cfg.Password = password
	}
	tlsEnable := false
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	if caCert != "" {
		certBytes, err := ioutil.ReadFile(caCert)
		if err != nil {
			return &Client{}, err
		}

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(certBytes)
		if ok {
			tlsConfig.RootCAs = caCertPool
		}
		tlsEnable = true
	}

	if cert != "" && key != "" {
		tlsCert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return &Client{}, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
		tlsEnable = true
	}

	if tlsEnable {
		cfg.TLS = tlsConfig
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		return &Client{}, err
	}

	return &Client{client, make(map[string]*Watch), sync.Mutex{}}, nil
}
