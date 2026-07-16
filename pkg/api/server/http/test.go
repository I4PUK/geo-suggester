package http

import (
	"context"
	"net/http"

	"github.com/example/geo-suggest/pkg/api/server"
	search_gateway "github.com/example/hotels-search/contracts/search-gateway/v1/langs/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TestHandler struct {
	service server.SearchService
}

func NewTestHandler(service server.SearchService) *TestHandler {
	return &TestHandler{service: service}
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	addResponseHeaders(w)
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "search-gateway.hotels-search-prod.svc:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		writeErrMessage(w, Error{Message: "can't connect to search-gateway " + err.Error()})
		return
	}
	searchClient := search_gateway.NewHotelSearchGatewayClient(conn)
	resp, err := searchClient.SearchHotels(ctx, &search_gateway.SearchHotelsRequest{
		GeoId: 2656875,
		Options: &search_gateway.SearchOptions{
			CheckInDate:  "2020-12-01",
			CheckOutDate: "2020-12-02",
			Guests: []*search_gateway.GuestRoom{
				{
					AdultCount: 1,
				},
			},
			B2B: &search_gateway.B2BData{
				IsBusinessTrip: false,
				LegalId:        0,
			},
			Flags: &search_gateway.Flags{
				BookingOffersEnabled: true,
			},
		},
	})
	if err != nil {
		writeErrMessage(w, Error{Message: "failed request to search-gateway " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	writeOKResponse(w, resp.HotelOffers)
}
