# SimpleRelic

SimpleRelic is a reporting library sending http metrics to NewRelic. In this (early)
stage it's tightly integrated with Gin framework. There is currently only one defined
metric (requests per endpoint), but should be easy to add user defined ones.

Apart from Gin framework the library does not have any external dependencies.

## Basic usage

In order to use default reporter that uses predefined metrics, you need to define
endpoints you want to monitor. To define an endpoint you need to give it a name
and a matcher function that tells whether the request url matches the endpoint.
A matcher function take an URL.Path as a parameter and returns bool (true if url
matches the endpoint, false otherwise).  

```
simplerelic.AddDefaultEndpoint(
    "log",
    func(urlPath string) bool { return strings.HasPrefix(urlPath, "/log") },
)

reporter, err := simplerelic.InitDefaultReporter(cfg.NewRelicName, cfg.NewRelicKey, cfg.DebugMode)
if err != nil {
    // handle error
}
reporter.Start()
```

## Roadmap

- The following metrics are (yet) to be implemented:
    - number of requests per status code (2xx, 3xx, 4xx, 5xx)
    - percentage of errors per endpoint
    - response time per endpoint
    - overall number of requests

- Making SimpleRelic independent from Gin framework.

Pull requests are welcome :)  

## Add an user defined metric

User defined metrics need to implement AppMetric interface.

```
type AppMetric interface {

    // Update the values on every requests (used in gin middleware)
    Update(c *gin.Context)

    // Clear the values (after they are reported)
    Clear()

    // ValueMap extracts all values to be reported to NewRelic
    ValueMap() map[string]float32
}
```

For example of a metric take a look at ReqPerEndpoint in metrics.go.

After you define your new metric you need to add it to the reporter.

```
reporter, err := simplerelic.InitDefaultReporter(cfg.NewRelicName, cfg.NewRelicKey, cfg.DebugMode)
if err != nil {
    // handle error
}
reporter.AddMetrics(NewUserDefinedMetric(simplerelic.DefaultEndpoints))
```

*Note that in this example we are passing default endpoints to a newly defined metric.
This is by no means neccessary. The new metric might have nothing to do with
endpoints or might use different endpoints that the rest of the metrics.*

## Custom NewRelic plugin

In case you add your own metrics and want to build dashboards and graphs for them,
you need to create your own NewRelic plugin. To report metrics to your own plugin
you just need to set the GUID (before creating the reporter).

```
simplerelic.Guid = "com.github.your_username.simplerelic"
```
