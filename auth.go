package sud

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

// 认证类型与请求头常量。
const (
	// authorizationType Authorization 认证类型固定值
	authorizationType = "Sud-Auth"

	// HeaderAuthorization 出站请求认证头
	HeaderAuthorization = "Authorization"

	// 入站回调验签时 Sud 携带的请求头
	HeaderSudAppID     = "Sud-AppId"
	HeaderSudTimestamp = "Sud-Timestamp"
	HeaderSudNonce     = "Sud-Nonce"
	HeaderSudSignature = "Sud-Signature"

	// DefaultNonceLen 默认随机串长度（与 gama 现网一致）
	DefaultNonceLen = 32

	nonceCharset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// Signer 出站请求签名器，同时提供入站回调验签。
// 签名串固定四行、每行以 \n 结尾（含最后一行）：
//
//	appID\ntimestamp\nnonce\nbody\n
//
// 签名值为 HmacSHA1(appSecret, 签名串) 的十六进制小写。
type Signer struct {
	appID     string
	appSecret string
	now       func() time.Time   // 可注入时钟
	nonceFunc func(n int) string // 可注入 nonce 生成（默认 NewNonce）
}

// NewSigner 创建签名器。opts 可注入时钟与 nonce 生成（测试用）。
func NewSigner(appID, appSecret string, opts ...SignerOption) *Signer {
	s := &Signer{
		appID:     appID,
		appSecret: appSecret,
		now:       time.Now,
		nonceFunc: NewNonce,
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// SignerOption 签名器可选项。
type SignerOption func(*Signer)

// WithClock 注入时钟（默认 time.Now）。
func WithClock(fn func() time.Time) SignerOption {
	return func(s *Signer) { s.now = fn }
}

// WithNonceFunc 注入 nonce 生成函数（默认 NewNonce）。
func WithNonceFunc(fn func(n int) string) SignerOption {
	return func(s *Signer) { s.nonceFunc = fn }
}

// SignContent 构造签名串（四行，含结尾换行）。
func (s *Signer) SignContent(timestamp, nonce string, body []byte) string {
	return s.appID + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
}

// Signature 计算签名值（十六进制小写）。
func (s *Signer) Signature(timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha1.New, []byte(s.appSecret))
	mac.Write([]byte(s.SignContent(timestamp, nonce, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Authorization 生成出站请求的 Authorization 头值，时间戳为当前毫秒。
func (s *Signer) Authorization(body []byte) string {
	timestamp := strconv.FormatInt(s.now().UnixMilli(), 10)
	nonce := s.nonceFunc(DefaultNonceLen)
	return fmt.Sprintf(`%s app_id="%s",timestamp="%s",nonce="%s",signature="%s"`,
		authorizationType, s.appID, timestamp, nonce, s.Signature(timestamp, nonce, body))
}

// VerifyCallback 校验 Sud 入站回调请求签名。
// 依次取 Sud-AppId / Sud-Timestamp / Sud-Nonce / Sud-Signature 四个头，
// 用相同签名串算法重算并常量时间比对；Sud-AppId 还必须与本 Signer 的 appID 一致。
func (s *Signer) VerifyCallback(header http.Header, body []byte) bool {
	appID := header.Get(HeaderSudAppID)
	if appID == "" || appID != s.appID {
		return false
	}
	timestamp := header.Get(HeaderSudTimestamp)
	nonce := header.Get(HeaderSudNonce)
	signature := header.Get(HeaderSudSignature)
	if timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	expect := s.Signature(timestamp, nonce, body)
	return hmac.Equal([]byte(expect), []byte(signature))
}

// NewNonce 生成 n 位随机字母数字串（crypto/rand）。
func NewNonce(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(nonceCharset)))
	for i := range b {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand 失败极罕见，退化用时间抖动避免 panic
			b[i] = nonceCharset[time.Now().UnixNano()%int64(len(nonceCharset))]
			continue
		}
		b[i] = nonceCharset[v.Int64()]
	}
	return string(b)
}
