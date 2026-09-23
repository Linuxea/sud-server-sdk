package api

import "context"

// OrderService 游戏内付费订单服务域。
type OrderService struct{ p Poster }

// NewOrderService 创建服务域。
func NewOrderService(p Poster) *OrderService { return &OrderService{p: p} }

// 订单状态常量。
const (
	OrderStatusCreated        = "CREATED"         // 订单创建完成，待执行
	OrderStatusExecuting      = "EXECUTING"       // 执行中
	OrderStatusExecuteFail    = "EXECUTE_FAIL"    // 执行失败
	OrderStatusExecuteSuccess = "EXECUTE_SUCCESS" // 执行成功
)

// CreateOrderReq 创建订单请求。cmd/value/payload 的取值见文档各游戏对应表。
type CreateOrderReq struct {
	OutOrderID string `json:"out_order_id"` // 商户自定义唯一订单号（64 字符以内）
	OutGroupID string `json:"out_group_id,omitempty"`
	MGID       string `json:"mg_id"`
	RoomID     string `json:"room_id"`
	Cmd        string `json:"cmd"`
	FromUID    string `json:"from_uid"`
	ToUID      string `json:"to_uid"`
	Value      int32  `json:"value"`
	Payload    any    `json:"payload,omitempty"` // 附加数据，游戏客户端透传时需原样透传
}

// CreateOrderResp 创建订单响应。
type CreateOrderResp struct {
	OutOrderID string `json:"out_order_id"`
	OrderID    string `json:"order_id"` // SUD 订单号
}

// OrderEntry 订单号对（商户订单号 ↔ SUD 订单号）。
type OrderEntry struct {
	OutOrderID string `json:"out_order_id"`
	OrderID    string `json:"order_id"`
}

// BatchCreateOrderEntry 批量创建订单的单条订单。
type BatchCreateOrderEntry struct {
	OutOrderID    string `json:"out_order_id"`
	Cmd           string `json:"cmd"`
	FromUID       string `json:"from_uid"`
	FromNickname  string `json:"from_nickname,omitempty"`
	FromAvatarURL string `json:"from_avatar_url,omitempty"`
	ToUID         string `json:"to_uid"`
	Value         int32  `json:"value"`
	Random        bool   `json:"random,omitempty"` // to_uid 不在玩家列表中时是否随机玩家触发
	Payload       any    `json:"payload,omitempty"`
}

// BatchCreateOrderReq 批量创建订单请求。
type BatchCreateOrderReq struct {
	MGID   string                  `json:"mg_id"`
	RoomID string                  `json:"room_id"`
	Orders []BatchCreateOrderEntry `json:"orders"`
}

// BatchCreateOrderResp 批量创建订单响应。
type BatchCreateOrderResp struct {
	Orders []OrderEntry `json:"orders"`
}

// QueryOrderReq 查询订单请求。OutOrderID 与 OrderID 不能同时为空，
// 同时存在时 OrderID 优先。
type QueryOrderReq struct {
	OutOrderID string `json:"out_order_id,omitempty"`
	OrderID    string `json:"order_id,omitempty"`
}

// QueryOrderResp 查询订单响应。
type QueryOrderResp struct {
	OrderID    string `json:"order_id"`
	OutOrderID string `json:"out_order_id"`
	OutGroupID string `json:"out_group_id,omitempty"`
	MGID       string `json:"mg_id"`
	RoomID     string `json:"room_id"`
	Cmd        string `json:"cmd"`
	FromUID    string `json:"from_uid"`
	ToUID      string `json:"to_uid"`
	Value      int32  `json:"value"`
	Payload    any    `json:"payload,omitempty"`
	Status     string `json:"status"` // 见 OrderStatus 常量
}

// Create 游戏内付费下单。
func (s *OrderService) Create(ctx context.Context, req CreateOrderReq) (*CreateOrderResp, error) {
	var resp CreateOrderResp
	if err := s.p.Post(ctx, string(KeyCreateOrder), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchCreate 批量创建订单。
func (s *OrderService) BatchCreate(ctx context.Context, req BatchCreateOrderReq) (*BatchCreateOrderResp, error) {
	var resp BatchCreateOrderResp
	if err := s.p.Post(ctx, string(KeyBatchCreateOrder), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Query 查询订单。
func (s *OrderService) Query(ctx context.Context, req QueryOrderReq) (*QueryOrderResp, error) {
	var resp QueryOrderResp
	if err := s.p.Post(ctx, string(KeyQueryOrder), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
