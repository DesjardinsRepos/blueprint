package specs

import (
	"github.com/blueprint-uservices/blueprint/blueprint/pkg/wiring"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/cart"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/catalogue"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/frontend"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/order"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/payment"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/queuemaster"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/shipping"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/user"
	"github.com/blueprint-uservices/blueprint/examples/sockshop/workload/workloadgen"
	"github.com/blueprint-uservices/blueprint/plugins/clientpool"
	"github.com/blueprint-uservices/blueprint/plugins/cmdbuilder"
	"github.com/blueprint-uservices/blueprint/plugins/goproc"
	"github.com/blueprint-uservices/blueprint/plugins/gotests"
	"github.com/blueprint-uservices/blueprint/plugins/grpc"
	"github.com/blueprint-uservices/blueprint/plugins/http"
	"github.com/blueprint-uservices/blueprint/plugins/linuxcontainer"
	"github.com/blueprint-uservices/blueprint/plugins/mongodb"
	"github.com/blueprint-uservices/blueprint/plugins/mysql"
	"github.com/blueprint-uservices/blueprint/plugins/opentelemetry"
	"github.com/blueprint-uservices/blueprint/plugins/retries"
	"github.com/blueprint-uservices/blueprint/plugins/simple"
	"github.com/blueprint-uservices/blueprint/plugins/thrift"
	"github.com/blueprint-uservices/blueprint/plugins/workflow"
	"github.com/blueprint-uservices/blueprint/plugins/workload"
	"github.com/blueprint-uservices/blueprint/plugins/zipkin"
)

// Comparison specs for measuring performance impact of different architectural choices
// Each spec varies along three dimensions:
// 1. RPC Framework: gRPC vs Thrift
// 2. Tracing: Zipkin enabled vs disabled
// 3. Architecture: Microservices vs Monolith

// 1. gRPC + Zipkin + Microservices (baseline)
var Cmp_gRPC_Zipkin_Micro = cmdbuilder.SpecOption{
	Name:        "cmp_grpc_zipkin_micro",
	Description: "Microservices with gRPC and Zipkin tracing",
	Build:       makeComparisonSpec(true, true, true),
}

// 2. Thrift + Zipkin + Microservices
var Cmp_Thrift_Zipkin_Micro = cmdbuilder.SpecOption{
	Name:        "cmp_thrift_zipkin_micro",
	Description: "Microservices with Thrift and Zipkin tracing",
	Build:       makeComparisonSpec(false, true, true),
}

// 3. gRPC + NoZipkin + Microservices
var Cmp_gRPC_NoZipkin_Micro = cmdbuilder.SpecOption{
	Name:        "cmp_grpc_nozipkin_micro",
	Description: "Microservices with gRPC, no tracing",
	Build:       makeComparisonSpec(true, false, true),
}

// 4. Thrift + NoZipkin + Microservices
var Cmp_Thrift_NoZipkin_Micro = cmdbuilder.SpecOption{
	Name:        "cmp_thrift_nozipkin_micro",
	Description: "Microservices with Thrift, no tracing",
	Build:       makeComparisonSpec(false, false, true),
}

// 5. Zipkin + Monolith
var Cmp_Zipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_zipkin_mono",
	Description: "Monolith with Zipkin tracing",
	Build:       makeComparisonSpec(true, true, false),
}

// 7. NoZipkin + Monolith
var Cmp_NoZipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_nozipkin_mono",
	Description: "Monolith with no tracing",
	Build:       makeComparisonSpec(true, false, false),
}


// Helper function to generate comparison specs
func makeComparisonSpec(useGRPC bool, useZipkin bool, useMicroservices bool) func(spec wiring.WiringSpec) ([]string, error) {
	return func(spec wiring.WiringSpec) ([]string, error) {
		// Define the trace collector if needed
		var trace_collector string
		if useZipkin {
			trace_collector = zipkin.Collector(spec, "zipkin")
		}

		// Modifiers that will be applied to all services
		applyDefaults := func(serviceName string, useHTTP ...bool) {

			// Add tracing first (if enabled)
			if useZipkin {
				opentelemetry.Instrument(spec, serviceName, trace_collector)
			}

			// For microservices, deploy to separate processes with RPC
			if useMicroservices {
				// Apply modifiers in correct order: tracing -> retries -> pooling -> RPC -> process -> container
				
				// RPC-related modifiers
				retries.AddRetries(spec, serviceName, 3)
				clientpool.Create(spec, serviceName, 10)
				
				// Choose RPC framework
				if len(useHTTP) > 0 && useHTTP[0] {
					http.Deploy(spec, serviceName)
				} else if useGRPC {
					grpc.Deploy(spec, serviceName)
				} else {
					thrift.Deploy(spec, serviceName)
				}
				
				goproc.Deploy(spec, serviceName)
				linuxcontainer.Deploy(spec, serviceName)
				
				// Add tests (each service can be tested independently in microservices)
				gotests.Test(spec, serviceName)
			}
		}

		user_db := mongodb.Container(spec, "user_db")
		user_service := workflow.Service[user.UserService](spec, "user_service", user_db)
		applyDefaults(user_service)

		payment_service := workflow.Service[payment.PaymentService](spec, "payment_service", "500")
		applyDefaults(payment_service)

		cart_db := mongodb.Container(spec, "cart_db")
		cart_service := workflow.Service[cart.CartService](spec, "cart_service", cart_db)
		applyDefaults(cart_service)

		shipqueue := simple.Queue(spec, "shipping_queue")
		shipdb := mongodb.Container(spec, "shipping_db")
		shipping_service := workflow.Service[shipping.ShippingService](spec, "shipping_service", shipqueue, shipdb)
		applyDefaults(shipping_service)

		// Deploy queue master to the same process as the shipping service
		queue_master := workflow.Service[queuemaster.QueueMaster](spec, "queue_master", shipqueue, shipping_service)
		if useMicroservices {
			goproc.AddToProcess(spec, "shipping_proc", queue_master)
		}

		order_db := mongodb.Container(spec, "order_db")
		order_service := workflow.Service[order.OrderService](spec, "order_service", user_service, cart_service, payment_service, shipping_service, order_db)
		applyDefaults(order_service)

		catalogue_db := mysql.Container(spec, "catalogue_db")
		catalogue_service := workflow.Service[catalogue.CatalogueService](spec, "catalogue_service", catalogue_db)
		applyDefaults(catalogue_service)

		frontend_service := workflow.Service[frontend.Frontend](spec, "frontend", user_service, catalogue_service, cart_service, order_service)
		
		// Apply modifiers and deployment based on architecture
		if useMicroservices {
			// Microservices: frontend with tracing, HTTP, retries, pooling, separate process/container
			// Apply in correct order
			if useZipkin {
				opentelemetry.Instrument(spec, frontend_service, trace_collector)
			}
			retries.AddRetries(spec, frontend_service, 3)
			clientpool.Create(spec, frontend_service, 10)
			http.Deploy(spec, frontend_service)
			frontend_proc := goproc.Deploy(spec, frontend_service)
			linuxcontainer.Deploy(spec, frontend_proc)
			gotests.Test(spec, frontend_service)
			
			wlgen := workload.Generator[workloadgen.SimpleWorkload](spec, "wlgen", frontend_service)
			return []string{frontend_proc + "_ctr", wlgen, "gotests"}, nil
		} else {
			// Monolith: frontend with HTTP for external access, deployed as single process
			// Internal services use direct calls (no RPC) since they have no Deploy modifiers
			// Note: Dont apply tracing to frontend in monolith - causes conflicts with HTTP
			// Backend services already have tracing from applyDefaults
			http.Deploy(spec, frontend_service)
			gotests.Test(spec, frontend_service)
			
			// Deploy to single process and container (goproc.Deploy bundles all dependencies)
			frontend_proc := goproc.Deploy(spec, frontend_service)
			linuxcontainer.Deploy(spec, frontend_proc)
			
			wlgen := workload.Generator[workloadgen.SimpleWorkload](spec, "wlgen", frontend_service)
			return []string{frontend_proc + "_ctr", wlgen, "gotests"}, nil
		}
	}
}
