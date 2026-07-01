package instrumentfixture

import "context"

type Service struct{}

func (s *Service) Do(ctx context.Context) error {
	return nil
}
