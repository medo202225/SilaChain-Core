package logging

type AuditLogger struct {
	logger *Logger
}

func NewAudit() *AuditLogger {
	return &AuditLogger{logger: New()}
}

func (a *AuditLogger) Event(action string, fields Field) {
	if a == nil || a.logger == nil {
		return
	}
	a.logger.Info("audit:"+action, fields)
}
