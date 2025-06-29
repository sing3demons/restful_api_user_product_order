package logger

// LoggerActionEnum as string constants
type LoggerActionEnum string

const (
	CONSUMING     LoggerActionEnum = "[CONSUMING]"
	PRODUCING     LoggerActionEnum = "[PRODUCING]"
	APP_LOGIC     LoggerActionEnum = "[APP_LOGIC]"
	HTTP_REQUEST  LoggerActionEnum = "[HTTP_REQUEST]"
	HTTP_RESPONSE LoggerActionEnum = "[HTTP_RESPONSE]"
	DB_REQUEST    LoggerActionEnum = "[DB_REQUEST]"
	DB_RESPONSE   LoggerActionEnum = "[DB_RESPONSE]"
	EXCEPTION     LoggerActionEnum = "[EXCEPTION]"
	INBOUND       LoggerActionEnum = "[INBOUND]"
	OUTBOUND      LoggerActionEnum = "[OUTBOUND]"
	SYSTEM        LoggerActionEnum = "[SYSTEM]"
	PRODUCED      LoggerActionEnum = "[PRODUCED]"
)

// DBActionEnum as string constants
type DBActionEnum string

const (
	QUERY  DBActionEnum = "QUERY"
	INSERT DBActionEnum = "INSERT"
	UPDATE DBActionEnum = "UPDATE"
	DELETE DBActionEnum = "DELETE"
)

// LoggerAction struct
type LoggerAction struct {
	Action            string
	ActionDescription string
	SubAction         string
}

// LoggerAction methods as functions
func NewConsuming(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(CONSUMING), ActionDescription: actionDescription, SubAction: subAction}
}

func NewProducing(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(PRODUCING), ActionDescription: actionDescription, SubAction: subAction}
}

func NewProduced(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(PRODUCED), ActionDescription: actionDescription, SubAction: subAction}
}

func NewDBRequest(operation DBActionEnum, actionDescription string) LoggerAction {
	return LoggerAction{Action: string(DB_REQUEST), ActionDescription: actionDescription, SubAction: string(operation)}
}

func NewDBResponse(operation DBActionEnum, actionDescription string) LoggerAction {
	return LoggerAction{Action: string(DB_RESPONSE), ActionDescription: actionDescription, SubAction: string(operation)}
}

func NewAppLogic(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(APP_LOGIC), ActionDescription: actionDescription, SubAction: subAction}
}

func NewHTTPRequest(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(HTTP_REQUEST), ActionDescription: actionDescription, SubAction: subAction}
}

func NewHTTPResponse(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(HTTP_RESPONSE), ActionDescription: actionDescription, SubAction: subAction}
}

func NewException(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(EXCEPTION), ActionDescription: actionDescription, SubAction: subAction}
}

func NewInbound(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(INBOUND), ActionDescription: actionDescription, SubAction: subAction}
}

func NewOutbound(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(OUTBOUND), ActionDescription: actionDescription, SubAction: subAction}
}

func NewSystem(actionDescription, subAction string) LoggerAction {
	return LoggerAction{Action: string(SYSTEM), ActionDescription: actionDescription, SubAction: subAction}
}
