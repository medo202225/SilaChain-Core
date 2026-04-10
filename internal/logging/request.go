package logging

func RequestFields(method string, path string, remoteAddr string) Field {
	return Field{
		"method":      method,
		"path":        path,
		"remote_addr": remoteAddr,
	}
}
