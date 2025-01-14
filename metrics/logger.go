package metrics

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// InitLogger configures the global logger
func InitLogger() {
	// Set logger output format
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set log output to both file and stdout
	logFile, err := os.OpenFile("logs/transactions.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
	}

	// Set log level based on DEBUG_MODE
	if os.Getenv("DEBUG_MODE") == "true" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
}

// LogInfo logs info level messages
func LogInfo(msg string) {
	log.Info(msg)
}

// LogError logs error level messages
func LogError(err error) {
	log.Error(err)
}

// LogDebug logs debug level messages
func LogDebug(msg string) {
	log.Debug(msg)
}

// RecordTransaction records transaction metrics
func RecordTransaction(txType, status string, duration float64) {
	log.WithFields(logrus.Fields{
		"type":     txType,
		"status":   status,
		"duration": duration,
	}).Info("Transaction processed")
}

// RecordError records error metrics
func RecordError(txType, errType string) {
	log.WithFields(logrus.Fields{
		"type":  txType,
		"error": errType,
	}).Error("Transaction error")
}

// GetLogger returns the logger instance
func GetLogger() *logrus.Logger {
	return log
}
