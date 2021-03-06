package logging

import "time"
import "fmt"
import "github.com/cloudfoundry-community/firehose-to-syslog/syslog"
import "github.com/Sirupsen/logrus"

type LoggingSyslog struct {
	Logger           *syslog.Logger
	LogrusLogger     *logrus.Logger
	syslogServer     string
	debugFlag        bool
	logFormatterType string
	syslogProtocol   string
}

func NewLoggingSyslog(SyslogServerFlag string, SysLogProtocolFlag string, LogFormatterFlag string, DebugFlag bool) Logging {
	return &LoggingSyslog{
		LogrusLogger:     logrus.New(),
		syslogServer:     SyslogServerFlag,
		logFormatterType: LogFormatterFlag,
		syslogProtocol:   SysLogProtocolFlag,
		debugFlag:        DebugFlag,
	}

}

func (l *LoggingSyslog) Connect() bool {
	l.LogrusLogger.Formatter = GetLogFormatter(l.logFormatterType)

	connectTimeout := time.Duration(10) * time.Second
	writeTimeout := time.Duration(5) * time.Second
	logger, err := syslog.Dial("doppler", l.syslogProtocol, l.syslogServer, nil /*tls cert*/, connectTimeout, writeTimeout, 0 /*tcp max line length*/)
	if err != nil {
		LogError("Could not connect to syslog endpoint", err)
		return false
	} else {
		LogStd(fmt.Sprintf("Connected to syslog endpoint %s://%s", l.syslogProtocol, l.syslogServer), l.debugFlag)
		l.Logger = logger
		return true
	}
}

func (l *LoggingSyslog) ShipEvents(eventFields map[string]interface{}, aMessage string) {
	// remove structured metadata prefixed fields in the message if it was added
	var sds string
	if eventFields["rfc5424_structureddata"] != nil {
		sds = eventFields["rfc5424_structureddata"].(string)
		delete(eventFields, "rfc5424_structureddata")
	}

	entry := l.LogrusLogger.WithFields(eventFields)
	entry.Message = aMessage
	formatted, _ := entry.String()

	//fmt.Fprintf(os.Stdout, "ShipEvents [%s] %s -| %s", aMessage, eventFields["event_type"], formatted)
	//TODO debug log of some kind?

	packet := syslog.Packet{
		Severity: syslog.SevInfo,
		Facility: syslog.LogLocal5,
		Hostname: "dopplerhostname", //TODO could get local machine name
		Tag:      "pcflog",          //TODO could get proc id - doppler[pid]
		//TODO on UDP it will be truncated to 1K
		//Time: eventFields["timestamp"],
		Time:           time.Now(),
		StructuredData: sds,       //[xxx yy="zz" uu="tt"][other@123 code="abc"]
		Message:        formatted, //For LogMessage, the stdout/stderr will be in "msg:" which comes from Logrus entry.Message
	}

	l.Logger.Write(packet)

}
