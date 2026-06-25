package main

import (
	"fmt"
)

// ========================================
// 1. 错误码定义
// ========================================

// 错误码规范：
// 1xxxx - 通用错误
// 2xxxx - 用户相关
// 3xxxx - 订单相关
// 4xxxx - 支付相关

const (
	// 通用错误 1xxxx
	CodeSuccess       = 0
	CodeUnknown       = 10000
	CodeInvalidParam  = 10001
	CodeUnauthorized  = 10002
	CodeForbidden     = 10003
	CodeNotFound      = 10004
	CodeConflict      = 10005
	CodeRateLimited   = 10006

	// 用户错误 2xxxx
	CodeUserNotFound    = 20001
	CodeUserExists      = 20002
	CodePasswordWrong   = 20003
	CodeTokenExpired    = 20004
	CodeTokenInvalid    = 20005

	// 订单错误 3xxxx
	CodeOrderNotFound   = 30001
	CodeOrderPaid       = 30002
	CodeOrderCancelled  = 30003
	CodeStockNotEnough  = 30004

	// 支付错误 4xxxx
	CodePayFailed       = 40001
	CodePayTimeout      = 40002
	CodePayAmountWrong  = 40003
)

// ========================================
// 2. 错误码映射
// ========================================

var codeMessages = map[int]string{
	CodeSuccess:       "success",
	CodeUnknown:       "unknown error",
	CodeInvalidParam:  "invalid parameter",
	CodeUnauthorized:  "unauthorized",
	CodeForbidden:     "forbidden",
	CodeNotFound:      "not found",
	CodeConflict:      "conflict",
	CodeRateLimited:   "rate limited",

	CodeUserNotFound:   "user not found",
	CodeUserExists:     "user already exists",
	CodePasswordWrong:  "wrong password",
	CodeTokenExpired:   "token expired",
	CodeTokenInvalid:   "token invalid",

	CodeOrderNotFound:  "order not found",
	CodeOrderPaid:      "order already paid",
	CodeOrderCancelled: "order cancelled",
	CodeStockNotEnough: "stock not enough",

	CodePayFailed:      "payment failed",
	CodePayTimeout:     "payment timeout",
	CodePayAmountWrong: "payment amount wrong",
}

// GetMessage 获取错误消息
func GetMessage(code int) string {
	if msg, ok := codeMessages[code]; ok {
		return msg
	}
	return "unknown error"
}

// ========================================
// 3. HTTP 状态码映射
// ========================================

func codeToHTTPStatus(code int) int {
	switch {
	case code == CodeSuccess:
		return 200
	case code >= 10000 && code < 20000:
		// 通用错误
		switch code {
		case CodeInvalidParam:
			return 400
		case CodeUnauthorized, CodeTokenExpired, CodeTokenInvalid:
			return 401
		case CodeForbidden:
			return 403
		case CodeNotFound:
			return 404
		case CodeConflict:
			return 409
		case CodeRateLimited:
			return 429
		default:
			return 500
		}
	case code >= 20000 && code < 30000:
		// 用户错误
		return 400
	case code >= 30000 && code < 40000:
		// 订单错误
		return 400
	case code >= 40000 && code < 50000:
		// 支付错误
		return 400
	default:
		return 500
	}
}

// ========================================
// 4. API 响应结构
// ========================================

// APIResponse 统一响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(data interface{}) APIResponse {
	return APIResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	}
}

// Error 错误响应
func Error(code int, message string) APIResponse {
	if message == "" {
		message = GetMessage(code)
	}
	return APIResponse{
		Code:    code,
		Message: message,
	}
}

// ErrorWithDetail 带详情的错误响应
func ErrorWithDetail(code int, message string, detail interface{}) APIResponse {
	return APIResponse{
		Code:    code,
		Message: message,
		Data:    detail,
	}
}

// ========================================
// 5. 业务错误
// ========================================

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ToResponse 转换为 API 响应
func (e *AppError) ToResponse() APIResponse {
	return Error(e.Code, e.Message)
}

// ========================================
// 6. 使用示例
// ========================================

func getUser(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, &AppError{
			Code:    CodeInvalidParam,
			Message: "user id is required",
		}
	}

	if id == "999" {
		return nil, &AppError{
			Code:    CodeUserNotFound,
			Message: "user not found",
		}
	}

	return map[string]interface{}{
		"id":   id,
		"name": "Alice",
	}, nil
}

func createOrder(userID, productID string) (map[string]interface{}, error) {
	if userID == "" || productID == "" {
		return nil, &AppError{
			Code:    CodeInvalidParam,
			Message: "user_id and product_id are required",
		}
	}

	// 模拟库存不足
	if productID == "out-of-stock" {
		return nil, &AppError{
			Code:    CodeStockNotEnough,
			Message: "product is out of stock",
		}
	}

	return map[string]interface{}{
		"order_id":   "order-123",
		"user_id":    userID,
		"product_id": productID,
		"status":     "created",
	}, nil
}

func main() {
	// ========================================
	// 1. 成功响应
	// ========================================
	fmt.Println("=== 成功响应 ===")
	user, err := getUser("1")
	if err != nil {
		var appErr *AppError
		if ae, ok := err.(*AppError); ok {
			appErr = ae
		}
		if appErr != nil {
			resp := appErr.ToResponse()
			fmt.Printf("HTTP Status: %d\n", codeToHTTPStatus(appErr.Code))
			fmt.Printf("Response: %+v\n", resp)
		}
	} else {
		resp := Success(user)
		fmt.Printf("Response: %+v\n", resp)
	}

	// ========================================
	// 2. 用户不存在
	// ========================================
	fmt.Println("\n=== 用户不存在 ===")
	_, err = getUser("999")
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			resp := appErr.ToResponse()
			fmt.Printf("HTTP Status: %d\n", codeToHTTPStatus(appErr.Code))
			fmt.Printf("Response: %+v\n", resp)
		}
	}

	// ========================================
	// 3. 创建订单 - 库存不足
	// ========================================
	fmt.Println("\n=== 库存不足 ===")
	_, err = createOrder("user-1", "out-of-stock")
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			resp := appErr.ToResponse()
			fmt.Printf("HTTP Status: %d\n", codeToHTTPStatus(appErr.Code))
			fmt.Printf("Response: %+v\n", resp)
		}
	}

	// ========================================
	// 4. 创建订单 - 成功
	// ========================================
	fmt.Println("\n=== 创建订单成功 ===")
	order, err := createOrder("user-1", "product-1")
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			resp := appErr.ToResponse()
			fmt.Printf("Response: %+v\n", resp)
		}
	} else {
		resp := Success(order)
		fmt.Printf("Response: %+v\n", resp)
	}

	// ========================================
	// 5. 错误码查询
	// ========================================
	fmt.Println("\n=== 错误码查询 ===")
	codes := []int{CodeSuccess, CodeUserNotFound, CodeStockNotEnough, CodePayTimeout}
	for _, code := range codes {
		fmt.Printf("Code %d: %s (HTTP %d)\n", code, GetMessage(code), codeToHTTPStatus(code))
	}
}
