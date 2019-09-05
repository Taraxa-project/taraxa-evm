// Hook go-metrics into expvar
// on any /debug/metrics request, load all vars from the registry into expvar, and execute regular expvar handler
package exp

// Exp will register an expvar powered metrics handler with http.DefaultServeMux on "/debug/vars"

// ExpHandler will return an expvar powered metrics handler.

