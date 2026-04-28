//go:build ignore

package sample

type service struct{}

func (service) Process(string) error { return nil }

func Run() {
	svc := service{}
	var _ = svc.Process("data")
}
