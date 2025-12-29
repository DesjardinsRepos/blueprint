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

// 5. gRPC + Zipkin + Monolith
var Cmp_gRPC_Zipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_grpc_zipkin_mono",
	Description: "Monolith with gRPC and Zipkin tracing",
	Build:       makeComparisonSpec(true, true, false),
}

// 6. Thrift + Zipkin + Monolith
var Cmp_Thrift_Zipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_thrift_zipkin_mono",
	Description: "Monolith with Thrift and Zipkin tracing",
	Build:       makeComparisonSpec(false, true, false),
}

// 7. gRPC + NoZipkin + Monolith
var Cmp_gRPC_NoZipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_grpc_nozipkin_mono",
	Description: "Monolith with gRPC, no tracing",
	Build:       makeComparisonSpec(true, false, false),
}

// 8. Thrift + NoZipkin + Monolith
var Cmp_Thrift_NoZipkin_Mono = cmdbuilder.SpecOption{
	Name:        "cmp_thrift_nozipkin_mono",
	Description: "Monolith with Thrift, no tracing",
	Build:       makeComparisonSpec(false, false, false),
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
			// Golang-level modifiers that add functionality
			retries.AddRetries(spec, serviceName, 3)
			clientpool.Create(spec, serviceName, 10)
			
			// Add tracing if enabled
			if useZipkin {
				opentelemetry.Instrument(spec, serviceName, trace_collector)
			}
			
			// Choose RPC framework
			if len(useHTTP) > 0 && useHTTP[0] {
				http.Deploy(spec, serviceName)
			} else if useGRPC {
				grpc.Deploy(spec, serviceName)
			} else {
				thrift.Deploy(spec, serviceName)
			}

			// For microservices, deploy to separate processes and containers
			if useMicroservices {
				goproc.Deploy(spec, serviceName)
				linuxcontainer.Deploy(spec, serviceName)
			}

			// Also add to tests
			gotests.Test(spec, serviceName)
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
		applyDefaults(frontend_service, true) // Frontend always uses HTTP

		wlgen := workload.Generator[workloadgen.SimpleWorkload](spec, "wlgen", frontend_service)

		// Return different artifacts based on architecture
		if useMicroservices {
			return []string{"frontend_ctr", wlgen, "gotests"}, nil
		} else {
			// For monolith, everything runs in one container
			return []string{"frontend_proc", wlgen, "gotests"}, nil
		}
	}
}
