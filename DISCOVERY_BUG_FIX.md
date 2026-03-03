// BUG FIX: Discovery Value Type Conversion Issue
// 
// PROBLEM: Two different DiscoveredValue structs exist:
// 1. resource.DiscoveredValue (in discovery.go) 
// 2. store.DiscoveredValue (in store.go)
//
// The normalizeDiscoveredValue function receives a *resource.DiscoveredValue 
// but tries to cast it as map[string]any, which fails.
// This causes the Exists field (and others) to not be properly transferred.

// SOLUTION: Fix the normalizeDiscoveredValue function to handle both types properly

func normalizeDiscoveredValue(raw any) DiscoveredValue {
	dv := DiscoveredValue{
		Installed: false,
		Version:   "",
		Exists:    false,
		Value:     nil,
		Raw:       make(map[string]any),
	}

	// Handle *resource.DiscoveredValue directly
	if rdv, ok := raw.(*resource.DiscoveredValue); ok {
		dv.Installed = rdv.Installed
		dv.Version = rdv.Version
		dv.Exists = rdv.Exists
		dv.Value = rdv.Value
		
		// Convert Raw map from interface{} to any
		for k, v := range rdv.Raw {
			dv.Raw[k] = v
		}
		return dv
	}

	// Handle map[string]any (fallback for other cases)
	if m, ok := raw.(map[string]any); ok {
		dv.Raw = m
		if v, ok := m["Installed"].(bool); ok {
			dv.Installed = v
		}
		if v, ok := m["installed"].(bool); ok {
			dv.Installed = v
		}
		if v, ok := m["Version"].(string); ok {
			dv.Version = v
		}
		if v, ok := m["version"].(string); ok {
			dv.Version = v
		}
		if v, ok := m["Exists"].(bool); ok {
			dv.Exists = v
		}
		if v, ok := m["exists"].(bool); ok {
			dv.Exists = v
		}
		if v, ok := m["Value"]; ok {
			dv.Value = v
		}
	} else {
		// Best effort: keep raw as a single entry
		dv.Raw["value"] = raw
		dv.Value = raw
	}

	return dv
}