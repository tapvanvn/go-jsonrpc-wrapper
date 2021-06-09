package route

import (
	"encoding/json"
	"fmt"

	"github.com/tapvanvn/go-jsonrpc-wrapper/route/response"
	"github.com/tapvanvn/gorouter/v2"
)

//Unhandle handle unhandling route
func Unhandle(context *gorouter.RouteContext) {

	fmt.Println("cannot handle:", context.Path)
	responseData := response.Data{Success: false,
		ErrorCode: 0,
		Message:   "Route to nowhere",
		Data:      nil}

	if data, err := json.Marshal(responseData); err == nil {

		context.W.Write(data)
	}
	context.Handled = true
}

//Root handle root
func Root(context *gorouter.RouteContext) {

	fmt.Println(context.Action)

	response.Success(context, "ok")
}
