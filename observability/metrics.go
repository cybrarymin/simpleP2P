package observ

import "github.com/prometheus/client_golang/prometheus"

func promInit() error {
	err := prometheus.Register(nil)
	if err != nil {
		return err
	}
	return nil
}
