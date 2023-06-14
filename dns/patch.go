//go:build foss

package dns

import (
	D "github.com/miekg/dns"

	"clash-foss/component/trie"
	"clash-foss/context"
)

func withAAAAFilter() middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, r *D.Msg) (*D.Msg, error) {
			if r.Question[0].Qtype == D.TypeAAAA {
				return handleMsgWithEmptyAnswer(r), nil
			}

			return next(ctx, r)
		}
	}
}

func NewHandler(resolver *Resolver, mapper *ResolverEnhancer, hosts *trie.DomainTrie, ipv6 bool) D.Handler {
	h := newHandler(resolver, mapper)

	if hosts != nil {
		h = withHosts(hosts)(h)
	}

	if !ipv6 {
		h = withAAAAFilter()(h)
	}

	return D.HandlerFunc(func(writer D.ResponseWriter, msg *D.Msg) {
		r, err := h(context.NewDNSContext(msg), msg)

		if err != nil {
			r = msg
			r.SetRcode(msg, D.RcodeServerFailure)
		}

		_ = writer.WriteMsg(r)
	})
}
