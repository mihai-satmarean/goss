package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/goss-org/goss/system"
	"github.com/goss-org/goss/util"
)

// Discovery represents a resource that is discovered and registered for use in templates
type Discovery struct {
	Title    string                 `json:"title,omitempty" yaml:"title,omitempty"`
	Meta     meta                   `json:"meta,omitempty" yaml:"meta,omitempty"`
	id       string                 `json:"-" yaml:"-"`
	Register string                 `json:"register" yaml:"register"`
	Type     string                 `json:"type" yaml:"type"`
	Resource map[string]interface{} `json:"resource" yaml:"resource"`
	Skip     bool                   `json:"skip,omitempty" yaml:"skip,omitempty"`
}

const (
	DiscoveryResourceKey  = "discovery"
	DiscoveryResourceName = "Discovery"
)

func init() {
	registerResource(DiscoveryResourceKey, &Discovery{})
}

func (d *Discovery) ID() string {
	return d.id
}
func (d *Discovery) SetID(id string)  { d.id = id }
func (d *Discovery) SetSkip()         { d.Skip = true }
func (d *Discovery) TypeKey() string  { return DiscoveryResourceKey }
func (d *Discovery) TypeName() string { return DiscoveryResourceName }
func (d *Discovery) GetTitle() string { return d.Title }
func (d *Discovery) GetMeta() meta    { return d.Meta }
func (d *Discovery) GetRegister() string {
	return d.Register
}

// DiscoveredValue holds the discovered values for a specific discovery
type DiscoveredValue struct {
	Installed bool                   `json:"installed,omitempty" yaml:"installed,omitempty"`
	Version   string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Exists    bool                   `json:"exists,omitempty" yaml:"exists,omitempty"`
	Value     interface{}            `json:"value,omitempty" yaml:"value,omitempty"`
	Raw       map[string]interface{} `json:"raw,omitempty" yaml:"raw,omitempty"`
}

// discoveredValueStore is a package-level cache for discovered values
var discoveredValueStore = make(map[string]*DiscoveredValue)

// Validate performs discovery and returns the discovered values
func (d *Discovery) Validate(sys *system.System) []TestResult {
	ctx := context.WithValue(context.Background(), idKey{}, d.ID())
	skip := d.Skip

	if skip {
		return []TestResult{skipResult(d.TypeName(), d.ID(), d.Title, d.Meta, "discovery", time.Now())}
	}

	discovered := &DiscoveredValue{
		Raw: make(map[string]interface{}),
	}

	startTime := time.Now()
	result := TestResult{
		Successful:   true,
		ResourceType: d.TypeName(),
		ResourceId:   d.ID(),
		Property:     "discovery",
		Result:       SUCCESS,
		Title:        d.Title,
		Meta:         d.Meta,
		StartTime:    startTime,
	}

	// Perform discovery based on resource type
	switch d.Type {
	case "package":
		d.discoverPackage(ctx, sys, discovered)
	case "file":
		d.discoverFile(ctx, sys, discovered)
	case "command":
		d.discoverCommand(ctx, sys, discovered)
	case "service":
		d.discoverService(ctx, sys, discovered)
	case "port":
		d.discoverPort(ctx, sys, discovered)
	case "user":
		d.discoverUser(ctx, sys, discovered)
	case "group":
		d.discoverGroup(ctx, sys, discovered)
	case "process":
		d.discoverProcess(ctx, sys, discovered)
	case "http":
		d.discoverHTTP(ctx, sys, discovered)
	case "addr":
		d.discoverAddr(ctx, sys, discovered)
	case "dns":
		d.discoverDNS(ctx, sys, discovered)
	case "kernel-param":
		d.discoverKernelParam(ctx, sys, discovered)
	case "mount":
		d.discoverMount(ctx, sys, discovered)
	case "interface":
		d.discoverInterface(ctx, sys, discovered)
	default:
		result.Successful = false
		result.Result = FAIL
		result.Err = toValidateError(fmt.Errorf("unknown discovery type: %s", d.Type))
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		return []TestResult{result}
	}

	// Store the discovered value for later retrieval
	discoveredValueStore[d.ID()] = discovered

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	return []TestResult{result}
}

// GetDiscoveredValue retrieves the discovered value from the store
func (d *Discovery) GetDiscoveredValue() *DiscoveredValue {
	return discoveredValueStore[d.ID()]
}

func (d *Discovery) discoverPackage(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	name, ok := d.Resource["name"].(string)
	if !ok {
		return
	}

	pkg := sys.NewPackage(ctx, name, sys, util.Config{})
	installed, _ := pkg.Installed()
	discovered.Installed = installed

	if installed {
		versions, err := pkg.Versions()
		if err == nil && len(versions) > 0 {
			discovered.Version = versions[0]
		}
	}

	discovered.Raw["name"] = name
	discovered.Raw["installed"] = installed
	if discovered.Version != "" {
		discovered.Raw["version"] = discovered.Version
	}
}

func (d *Discovery) discoverFile(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	path, ok := d.Resource["path"].(string)
	if !ok {
		return
	}

	file := sys.NewFile(ctx, path, sys, util.Config{})
	exists, _ := file.Exists()
	discovered.Exists = exists

	discovered.Raw["path"] = path
	discovered.Raw["exists"] = exists

	if exists {
		if filetype, err := file.Filetype(); err == nil {
			discovered.Raw["filetype"] = filetype
		}
		if mode, err := file.Mode(); err == nil {
			discovered.Raw["mode"] = mode
		}
		if owner, err := file.Owner(); err == nil {
			discovered.Raw["owner"] = owner
		}
		if group, err := file.Group(); err == nil {
			discovered.Raw["group"] = group
		}
		if size, err := file.Size(); err == nil {
			discovered.Raw["size"] = size
		}
	}
}

func (d *Discovery) discoverCommand(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	command, ok := d.Resource["command"].(string)
	if !ok {
		return
	}

	cmd := sys.NewCommand(ctx, command, sys, util.Config{})
	exitStatus, _ := cmd.ExitStatus()
	
	discovered.Raw["command"] = command
	discovered.Raw["exit-status"] = exitStatus

	if stdout, err := cmd.Stdout(); err == nil {
		discovered.Raw["stdout"] = stdout
		discovered.Value = stdout
	}
	if stderr, err := cmd.Stderr(); err == nil {
		discovered.Raw["stderr"] = stderr
	}
}

func (d *Discovery) discoverService(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	name, ok := d.Resource["name"].(string)
	if !ok {
		return
	}

	service := sys.NewService(ctx, name, sys, util.Config{})
	enabled, _ := service.Enabled()
	running, _ := service.Running()

	discovered.Raw["name"] = name
	discovered.Raw["enabled"] = enabled
	discovered.Raw["running"] = running
	discovered.Exists = enabled || running
}

func (d *Discovery) discoverPort(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	port, ok := d.Resource["port"].(string)
	if !ok {
		return
	}

	portRes := sys.NewPort(ctx, port, sys, util.Config{})
	listening, _ := portRes.Listening()

	discovered.Raw["port"] = port
	discovered.Raw["listening"] = listening
	discovered.Exists = listening
}

func (d *Discovery) discoverUser(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	username, ok := d.Resource["username"].(string)
	if !ok {
		return
	}

	user := sys.NewUser(ctx, username, sys, util.Config{})
	exists, _ := user.Exists()
	discovered.Exists = exists

	discovered.Raw["username"] = username
	discovered.Raw["exists"] = exists

	if exists {
		if uid, err := user.UID(); err == nil {
			discovered.Raw["uid"] = uid
		}
		if gid, err := user.GID(); err == nil {
			discovered.Raw["gid"] = gid
		}
		if home, err := user.Home(); err == nil {
			discovered.Raw["home"] = home
		}
		if shell, err := user.Shell(); err == nil {
			discovered.Raw["shell"] = shell
		}
		if groups, err := user.Groups(); err == nil {
			discovered.Raw["groups"] = groups
		}
	}
}

func (d *Discovery) discoverGroup(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	groupname, ok := d.Resource["groupname"].(string)
	if !ok {
		return
	}

	group := sys.NewGroup(ctx, groupname, sys, util.Config{})
	exists, _ := group.Exists()
	discovered.Exists = exists

	discovered.Raw["groupname"] = groupname
	discovered.Raw["exists"] = exists

	if exists {
		if gid, err := group.GID(); err == nil {
			discovered.Raw["gid"] = gid
		}
	}
}

func (d *Discovery) discoverProcess(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	comm, ok := d.Resource["comm"].(string)
	if !ok {
		return
	}

	process := sys.NewProcess(ctx, comm, sys, util.Config{})
	running, _ := process.Running()

	discovered.Raw["comm"] = comm
	discovered.Raw["running"] = running
	discovered.Exists = running
}

func (d *Discovery) discoverHTTP(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	url, ok := d.Resource["url"].(string)
	if !ok {
		return
	}

	http := sys.NewHTTP(ctx, url, sys, util.Config{})
	
	discovered.Raw["url"] = url

	if status, err := http.Status(); err == nil {
		discovered.Raw["status"] = status
	}
	if headers, err := http.Headers(); err == nil {
		discovered.Raw["headers"] = headers
	}
}

func (d *Discovery) discoverAddr(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	address, ok := d.Resource["address"].(string)
	if !ok {
		return
	}

	addr := sys.NewAddr(ctx, address, sys, util.Config{})
	reachable, _ := addr.Reachable()

	discovered.Raw["address"] = address
	discovered.Raw["reachable"] = reachable
	discovered.Exists = reachable
}

func (d *Discovery) discoverDNS(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	host, ok := d.Resource["host"].(string)
	if !ok {
		return
	}

	dns := sys.NewDNS(ctx, host, sys, util.Config{})
	
	discovered.Raw["host"] = host

	if resolvable, err := dns.Resolvable(); err == nil {
		discovered.Raw["resolvable"] = resolvable
		discovered.Exists = resolvable
	}
	if addrs, err := dns.Addrs(); err == nil {
		discovered.Raw["addrs"] = addrs
	}
}

func (d *Discovery) discoverKernelParam(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	key, ok := d.Resource["key"].(string)
	if !ok {
		return
	}

	param := sys.NewKernelParam(ctx, key, sys, util.Config{})
	
	discovered.Raw["key"] = key

	if value, err := param.Value(); err == nil {
		discovered.Raw["value"] = value
		discovered.Value = value
	}
}

func (d *Discovery) discoverMount(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	mountpoint, ok := d.Resource["mountpoint"].(string)
	if !ok {
		return
	}

	mount := sys.NewMount(ctx, mountpoint, sys, util.Config{})
	exists, _ := mount.Exists()
	discovered.Exists = exists

	discovered.Raw["mountpoint"] = mountpoint
	discovered.Raw["exists"] = exists

	if exists {
		if source, err := mount.Source(); err == nil {
			discovered.Raw["source"] = source
		}
		if filesystem, err := mount.Filesystem(); err == nil {
			discovered.Raw["filesystem"] = filesystem
		}
		if opts, err := mount.Opts(); err == nil {
			discovered.Raw["opts"] = opts
		}
	}
}

func (d *Discovery) discoverInterface(ctx context.Context, sys *system.System, discovered *DiscoveredValue) {
	name, ok := d.Resource["name"].(string)
	if !ok {
		return
	}

	iface := sys.NewInterface(ctx, name, sys, util.Config{})
	exists, _ := iface.Exists()
	discovered.Exists = exists

	discovered.Raw["name"] = name
	discovered.Raw["exists"] = exists

	if exists {
		if addrs, err := iface.Addrs(); err == nil {
			discovered.Raw["addrs"] = addrs
		}
		if mtu, err := iface.MTU(); err == nil {
			discovered.Raw["mtu"] = mtu
		}
	}
}

