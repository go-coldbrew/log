/*
Package log provides a minimal interface for structured logging in services. ColdBrew uses this log package for all logs.
It provides a simple interface to log errors, warnings, info and debug messages.
It also provides a mechanism to add contextual information to logs.
available implementations of BaseLogger are in loggers package. You can also implement your own BaseLogger to use with this package.

# How To Use

The simplest way to use this package is by calling static log functions to report particular level (error/warning/info/debug)

	log.Error(...)
	log.Warn(...)
	log.Info(...)
	log.Debug(...)

You can also initialize a new logger by calling 'log.NewLogger' and passing a loggers.BaseLogger implementation (loggers package provides a number of pre built implementations)

	logger := log.NewLogger(gokit.NewLogger())
	logger.Info(ctx, "key", "value")

Note:

	Preferred logging output is in either logfmt or json format, so to facilitate these log function arguments should be in pairs of key-value

# Contextual Logs

log package uses context.Context to pass additional information to logs, you can use 'loggers.AddToLogContext' function to add additional information to logs. For example in access log from service

	{"@timestamp":"2018-07-30T09:58:18.262948679Z","caller":"http/http.go:66","error":null,"grpcMethod":"/AuthSvc.AuthService/Authenticate","level":"info","method":"POST","path":"/2.0/authenticate/","took":"1.356812ms","trace":"15592e1b-93df-11e8-bdfd-0242ac110002","transport":"http"}

we pass 'grpcMethod' from context, this information gets automatically added to all log calls called inside the service and makes debugging services much easier.
ColdBrew also generates a 'trace' ID per request, this can be used to trace an entire request path in logs.

this package is based on https://github.com/carousell/Orion/tree/master/utils/log
*/
package log
