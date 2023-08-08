package jaegerotel

type JaegerTracerProviderOption func(tp *tracerProvider)

func WithConfig(service, environment string) JaegerTracerProviderOption {
	return func(tp *tracerProvider) {
		tp.service = service
		tp.environment = environment
	}
}
