package main

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// ---------------------------------------------------------
// Process with go func incrementing counter each 2 seconds
// myapp_processed_ops_total += 1

var opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
	Name: "myapp_processed_ops_total",
	Help: "The total number of processed events",
})

func asyncCounterMetric() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}

// --------------------------------------------------------------------------------------------------------------------
// Process that get and calculate with external information metrics related with Chile from mindicador.cl each minute
// Topics: UF (clp) , Bitcoin (usd, clp), Unemployment-Rate (percent)

var chileIndicators = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "chile_indicators",
		Help: "Economy indicators of Chile",
	},
	[]string{"code", "unit"},
)

func asyncGetChileIndicators() {
	go func() {
		for {
			resp, err := http.Get("https://mindicador.cl/api")
			if err != nil {
				log.Fatalln(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			jsonMap := make(map[string]interface{})
			err = json.Unmarshal(body, &jsonMap)
			if err != nil {
				log.Fatalln(err)
			}

			ufVal := jsonMap["uf"].(map[string]interface{})["valor"].(float64)
			chileIndicators.WithLabelValues("uf", "clp").Set(ufVal)

			unemploymentRateVal := jsonMap["tasa_desempleo"].(map[string]interface{})["valor"].(float64)
			chileIndicators.WithLabelValues("unemployment-rate", "%").Set(unemploymentRateVal)

			bitcoinVal := jsonMap["bitcoin"].(map[string]interface{})["valor"].(float64)
			usdVal := jsonMap["dolar"].(map[string]interface{})["valor"].(float64)

			chileIndicators.WithLabelValues("bitcoin", "usd").Set(bitcoinVal)
			chileIndicators.WithLabelValues("bitcoin", "clp").Set(bitcoinVal * usdVal)

			time.Sleep(time.Minute)
		}
	}()
}

// --------------------------------------------------------------------------------------------------------------------
// When you consume this endpoint http://localhost:5000/random you generate a random response code and immediately this
// code is registered in prometheus metrics as random_response_handler_requests_total

var randomResponseStats = make(map[int]prometheus.Counter)

func randomResponseController(w http.ResponseWriter, r *http.Request) {
	responses := []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError}
	statusCode := responses[rand.Intn(len(responses))]

	if _, ok := randomResponseStats[statusCode]; !ok {

		randomResponseStats[statusCode] = promauto.NewCounter(prometheus.CounterOpts{
			Name: "random_response_handler_requests_total",
			Help: "Counter of get by HTTP status code.",
			ConstLabels: map[string]string{
				"method": http.MethodGet,
				"code":   strconv.Itoa(statusCode),
				"msg":    http.StatusText(statusCode),
			},
		})

	}

	randomResponseStats[statusCode].Inc()

	w.WriteHeader(statusCode)
	w.Write([]byte(strconv.Itoa(statusCode) + " - " + http.StatusText(statusCode)))
}

func main() {
	asyncCounterMetric()
	asyncGetChileIndicators()

	http.Handle("/random", http.HandlerFunc(randomResponseController))

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe("0.0.0.0:5000", nil)
}
