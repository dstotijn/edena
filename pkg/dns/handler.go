package dns

import (
	"context"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

func (srv *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	ctx := context.Background() // TODO: Introduce context on `srv`?

	if len(r.Question) == 0 {
		return
	}
	name := r.Question[0].Name

	reply := &dns.Msg{}
	_ = reply.SetReply(r)
	defer func() {
		err := w.WriteMsg(reply)
		if err != nil {
			srv.logger.Error("Failed to write DNS reply.", zap.Error(err))
		}
	}()

	if !dns.IsSubDomain(dns.Fqdn(srv.soaHostname), dns.Fqdn(name)) {
		return
	}

	reply.Authoritative = true

	switch r.Question[0].Qtype {
	case dns.TypeSOA:
		soa := &dns.SOA{
			Ns: dns.Fqdn(libdns.AbsoluteName("ns1", srv.soaHostname)),
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeSOA,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			Mbox:    libdns.AbsoluteName("hostmaster", srv.soaHostname),
			Serial:  1,
			Refresh: 86400,
			Retry:   7200,
			Expire:  3600000,
			Minttl:  3600,
		}
		reply.Answer = append(reply.Answer, soa)
	case dns.TypeNS:
		ns := &dns.NS{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    3600,
			},
			Ns: dns.Fqdn(libdns.AbsoluteName("ns1", srv.soaHostname)),
		}
		reply.Answer = append(reply.Answer, ns)
	default:
		recs, err := srv.GetRecords(ctx, name)
		if err != nil {
			srv.logger.Error("Failed to get records for zone.",
				zap.String("name", name),
				zap.Error(err),
			)
			return
		}
		for _, rec := range recs {
			if _, ok := dns.StringToType[rec.Type]; ok {
				rr, err := MessageFromRecord(name, rec)
				if err != nil {
					srv.logger.Error("Failed to parse message from record.", zap.Error(err))
					return
				}
				reply.Answer = append(reply.Answer, rr)
			}
		}
	}
}
