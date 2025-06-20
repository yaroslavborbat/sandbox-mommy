package rest

import scontrollerutil "github.com/yaroslavborbat/sandbox-mommy/pkg/controller/util"

var secrets = &podSecrets{}

type podSecrets struct {
	ca    []byte
	token string
}

func (s *podSecrets) load() error {
	if s.token == "" {
		token, err := scontrollerutil.GetServiceAccountToken()
		if err != nil {
			return err
		}
		s.token = token
	}
	if len(s.ca) == 0 {
		ca, err := scontrollerutil.GetCA()
		if err != nil {
			return err
		}
		s.ca = ca
	}

	return nil
}
