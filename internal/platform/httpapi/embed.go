package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// validateEmbedURL checks headers only; it never proxies remote page content.
func (s *server) validateEmbedURL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	target, err := url.Parse(strings.TrimSpace(input.URL))
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Hostname() == "" || blockedEmbedHost(target.Hostname()) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"embeddable": false, "reason": "请输入可公开访问的 http 或 https 地址"})
		return
	}
	client := &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || blockedEmbedHost(req.URL.Hostname()) {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodHead, target.String(), nil)
	response, err := client.Do(request)
	if err == nil && response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"embeddable": false, "reason": "无法连接目标网站，请确认地址可访问"})
		return
	}
	xfo := strings.ToUpper(response.Header.Get("X-Frame-Options"))
	csp := strings.ToLower(response.Header.Get("Content-Security-Policy"))
	if strings.Contains(xfo, "DENY") || strings.Contains(xfo, "SAMEORIGIN") {
		writeJSON(w, http.StatusOK, map[string]any{"embeddable": false, "reason": "目标网站设置了 X-Frame-Options，禁止被嵌入"})
		return
	}
	if strings.Contains(csp, "frame-ancestors") && !strings.Contains(csp, "frame-ancestors *") {
		writeJSON(w, http.StatusOK, map[string]any{"embeddable": false, "reason": "目标网站的内容安全策略禁止在此画布中嵌入"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"embeddable": true, "reason": "未发现禁止嵌入的响应头"})
}

func blockedEmbedHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" || host == "" {
		return true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if parsed, ok := netip.AddrFromSlice(ip); ok && (parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsUnspecified()) {
			return true
		}
	}
	return false
}
