package model

// ResourceContext holds resource usage context for a process
type ResourceContext struct {
  // Energy impact level: "", "Low", "Medium", "High"
  EnergyImpact string

  // Whether the process is preventing system sleep
  PreventsSleep bool

  // Thermal state: "", "Normal", "Moderate", "Heavy", "Trapping", "Sleeping"
  ThermalState string

  // CPU scheduling priority if throttled
  AppNapped bool
}
