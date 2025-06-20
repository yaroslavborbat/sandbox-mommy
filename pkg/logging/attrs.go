package logging

import "log/slog"

func SlogApp(app string) slog.Attr {
	return slog.String("app", app)
}

func SlogController(controller string) slog.Attr {
	return slog.String("controller", controller)
}

func SlogErr(err error) slog.Attr {
	if err == nil {
		return slog.String("err", "NO ERROR")
	}
	return slog.String("err", err.Error())
}

func SlogName(name string) slog.Attr {
	return slog.String("name", name)
}

func SlogNamespace(namespace string) slog.Attr {
	return slog.String("namespace", namespace)
}

func SlogStorage(storage string) slog.Attr {
	return slog.String("storage", storage)
}
