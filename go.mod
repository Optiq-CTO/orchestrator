module github.com/Optiq-CTO/orchestrator

go 1.24.7

require (
	github.com/Optiq-CTO/analyzer v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251222181119-0a764e51fe1b // indirect
)

replace github.com/Optiq-CTO/analyzer => ../analyzer

replace github.com/Optiq-CTO/creator => ../creator

replace github.com/Optiq-CTO/fetcher => ../fetcher

replace github.com/Optiq-CTO/publisher => ../publisher

replace github.com/Optiq-CTO/aicontext => ../aicontext
