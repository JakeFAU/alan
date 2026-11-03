package author

import "github.com/JakeFAU/alan/schema"

// ContinuationPort declares the ordinary local input selected by an opted-in
// runtime continuation slot. Reasons are application policy. Receive(port, nil)
// suffices when the agent only needs the authorization.
func ContinuationPort(obligation string, reasons ...string) Port {
	return Port{Peer: "local", Local: true, Obligation: obligation, Receive: schema.Record{
		"previous_operation_id": {Kind: "string", MaxBytes: 4096},
		"purpose":               {Kind: "string", MaxBytes: 64, Enum: reasons},
		"prepared_digest":       {Kind: "string", MaxBytes: 64},
		"result_digest":         {Kind: "string", MaxBytes: 64},
	}}
}
