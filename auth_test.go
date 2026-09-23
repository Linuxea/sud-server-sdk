package sud

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// golden 向量由 python hmac 独立计算（见阶段 2 记录），锁定签名串构造与算法不回归。
const (
	goldenAppID       = "1461564080052506636"
	goldenSecret      = "test-app-secret"
	goldenTS          = "1688464949817"
	goldenNonce       = "keVJLJTItd1VBtGT"
	goldenOutBody     = `{"app_id": "1461564080052506238","mg_id": "1461227817776713818","room_id": "9009"}`
	goldenOutSig      = "6c83c0c0d430f987fdc903dde6ed42acc4f9a277"
	goldenInBody      = `{"app_id": "1461564080052506238","mg_id": "1461227817776713818","room_id": "9009","round_id": "ce56b6lzi1a7-cehorlmy01pq-ckmfkba10iv7","currency_amount": "2", "timestamp": 1654079242000}`
	goldenInTS        = "1654079242000"
	goldenInSig       = "136cb2eeb55b0f642f57de8b044e8ae91bf01978"
	goldenSignContent = "1461564080052506636\n1688464949817\nkeVJLJTItd1VBtGT\n" + goldenOutBody + "\n"
)

func TestSignContent(t *testing.T) {
	s := NewSigner(goldenAppID, goldenSecret)
	if got := s.SignContent(goldenTS, goldenNonce, []byte(goldenOutBody)); got != goldenSignContent {
		t.Fatalf("签名串不匹配:\ngot  %q\nwant %q", got, goldenSignContent)
	}
}

func TestSignature(t *testing.T) {
	s := NewSigner(goldenAppID, goldenSecret)
	if got := s.Signature(goldenTS, goldenNonce, []byte(goldenOutBody)); got != goldenOutSig {
		t.Fatalf("签名值 got %s want %s", got, goldenOutSig)
	}
}

func TestAuthorization(t *testing.T) {
	// 注入固定时钟与 nonce，对完整 Authorization 头做 golden 断言
	fixedTime := time.UnixMilli(1688464949817)
	s := NewSigner(goldenAppID, goldenSecret,
		WithClock(func() time.Time { return fixedTime }),
		WithNonceFunc(func(int) string { return goldenNonce }),
	)
	got := s.Authorization([]byte(goldenOutBody))
	want := `Sud-Auth app_id="1461564080052506636",timestamp="1688464949817",nonce="keVJLJTItd1VBtGT",signature="` + goldenOutSig + `"`
	if got != want {
		t.Fatalf("Authorization:\ngot  %s\nwant %s", got, want)
	}
}

func callbackHeader(body string) http.Header {
	s := NewSigner(goldenAppID, goldenSecret)
	h := http.Header{}
	h.Set(HeaderSudAppID, goldenAppID)
	h.Set(HeaderSudTimestamp, goldenInTS)
	h.Set(HeaderSudNonce, goldenNonce)
	h.Set(HeaderSudSignature, s.Signature(goldenInTS, goldenNonce, []byte(body)))
	return h
}

func TestVerifyCallback(t *testing.T) {
	s := NewSigner(goldenAppID, goldenSecret)

	if !s.VerifyCallback(callbackHeader(goldenInBody), []byte(goldenInBody)) {
		t.Fatal("合法回调应验签通过")
	}
	// 固定签名 golden 向量
	h := http.Header{}
	h.Set(HeaderSudAppID, goldenAppID)
	h.Set(HeaderSudTimestamp, goldenInTS)
	h.Set(HeaderSudNonce, goldenNonce)
	h.Set(HeaderSudSignature, goldenInSig)
	if !s.VerifyCallback(h, []byte(goldenInBody)) {
		t.Fatal("文档示例签名应验签通过")
	}
	// 篡改 body
	if s.VerifyCallback(callbackHeader(goldenInBody), []byte(goldenInBody+" ")) {
		t.Fatal("篡改 body 应验签失败")
	}
	// 错误 appID
	h2 := callbackHeader(goldenInBody)
	h2.Set(HeaderSudAppID, "1461564080052506238")
	if s.VerifyCallback(h2, []byte(goldenInBody)) {
		t.Fatal("appID 不匹配应验签失败")
	}
	// 缺头
	h3 := http.Header{}
	h3.Set(HeaderSudAppID, goldenAppID)
	if s.VerifyCallback(h3, []byte(goldenInBody)) {
		t.Fatal("缺失请求头应验签失败")
	}
}

func TestNewNonce(t *testing.T) {
	n := NewNonce(32)
	if len(n) != 32 {
		t.Fatalf("nonce 长度 %d", len(n))
	}
	for _, c := range n {
		if !strings.ContainsRune(nonceCharset, c) {
			t.Fatalf("nonce 含非法字符 %q", c)
		}
	}
	if NewNonce(16) == NewNonce(16) {
		t.Fatal("两次 nonce 不应相同")
	}
}
