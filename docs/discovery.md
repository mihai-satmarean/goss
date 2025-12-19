# Discovery Feature

The discovery feature allows you to conditionally run tests based on the state of your system. This is useful when you want to run the same Goss configuration across different servers with different setups.

## Overview

Discovery resources are executed before the main test suite and their results are made available to templates through the `.Discovered` variable. This allows you to:

1. Check if a package or service exists before running related tests
2. Capture system state (versions, configurations, etc.)
3. Make tests portable across different environments

## Syntax

### Basic Discovery

```yaml
discovery:
  auditd_check:
    type: package
    resource:
      name: auditd
    register: auditd_installed
```

### Using Discovered Values in Tests

```yaml
# Only run this test if auditd is installed
{{ if .Discovered.auditd_installed.Installed }}
file:
  /etc/audit/auditd.conf:
    exists: true
    contains:
      - 'log_file = /var/log/audit/audit.log'
{{ end }}
```

## Discovery Types

Discovery supports all standard Goss resource types:

### Package Discovery

```yaml
discovery:
  check_nginx:
    type: package
    resource:
      name: nginx
    register: nginx_info
```

Available fields:
- `.Discovered.nginx_info.Installed` - boolean
- `.Discovered.nginx_info.Version` - string (if installed)
- `.Discovered.nginx_info.Raw` - map with all package information

### File Discovery

```yaml
discovery:
  check_config:
    type: file
    resource:
      path: /etc/myapp/config.json
    register: config_file
```

Available fields:
- `.Discovered.config_file.Exists` - boolean
- `.Discovered.config_file.Raw.filetype` - string
- `.Discovered.config_file.Raw.mode` - string
- `.Discovered.config_file.Raw.owner` - string
- `.Discovered.config_file.Raw.size` - int

### Command Discovery

```yaml
discovery:
  get_version:
    type: command
    resource:
      command: "myapp --version"
    register: app_version
```

Available fields:
- `.Discovered.app_version.Value` - stdout of the command
- `.Discovered.app_version.Raw.exit-status` - int
- `.Discovered.app_version.Raw.stdout` - string
- `.Discovered.app_version.Raw.stderr` - string

### Service Discovery

```yaml
discovery:
  check_service:
    type: service
    resource:
      name: sshd
    register: ssh_service
```

Available fields:
- `.Discovered.ssh_service.Exists` - boolean (true if enabled or running)
- `.Discovered.ssh_service.Raw.enabled` - boolean
- `.Discovered.ssh_service.Raw.running` - boolean

### Port Discovery

```yaml
discovery:
  check_port:
    type: port
    resource:
      port: "tcp:80"
    register: web_port
```

Available fields:
- `.Discovered.web_port.Exists` - boolean (true if listening)
- `.Discovered.web_port.Raw.listening` - boolean

### User Discovery

```yaml
discovery:
  check_user:
    type: user
    resource:
      username: appuser
    register: app_user
```

Available fields:
- `.Discovered.app_user.Exists` - boolean
- `.Discovered.app_user.Raw.uid` - int
- `.Discovered.app_user.Raw.gid` - int
- `.Discovered.app_user.Raw.home` - string
- `.Discovered.app_user.Raw.shell` - string
- `.Discovered.app_user.Raw.groups` - []string

### Group Discovery

```yaml
discovery:
  check_group:
    type: group
    resource:
      groupname: docker
    register: docker_group
```

Available fields:
- `.Discovered.docker_group.Exists` - boolean
- `.Discovered.docker_group.Raw.gid` - int

### Process Discovery

```yaml
discovery:
  check_process:
    type: process
    resource:
      comm: nginx
    register: nginx_process
```

Available fields:
- `.Discovered.nginx_process.Exists` - boolean (true if running)
- `.Discovered.nginx_process.Raw.running` - boolean

### HTTP Discovery

```yaml
discovery:
  check_api:
    type: http
    resource:
      url: http://localhost:8080/health
    register: api_health
```

Available fields:
- `.Discovered.api_health.Raw.status` - int
- `.Discovered.api_health.Raw.headers` - map[string][]string

### DNS Discovery

```yaml
discovery:
  check_dns:
    type: dns
    resource:
      host: example.com
    register: dns_lookup
```

Available fields:
- `.Discovered.dns_lookup.Exists` - boolean (true if resolvable)
- `.Discovered.dns_lookup.Raw.resolvable` - boolean
- `.Discovered.dns_lookup.Raw.addrs` - []string

### Kernel Parameter Discovery

```yaml
discovery:
  check_kernel_param:
    type: kernel-param
    resource:
      key: kernel.shmmax
    register: shmmax
```

Available fields:
- `.Discovered.shmmax.Value` - string
- `.Discovered.shmmax.Raw.value` - string

### Mount Discovery

```yaml
discovery:
  check_mount:
    type: mount
    resource:
      mountpoint: /data
    register: data_mount
```

Available fields:
- `.Discovered.data_mount.Exists` - boolean
- `.Discovered.data_mount.Raw.source` - string
- `.Discovered.data_mount.Raw.filesystem` - string
- `.Discovered.data_mount.Raw.opts` - []string

### Interface Discovery

```yaml
discovery:
  check_interface:
    type: interface
    resource:
      name: eth0
    register: eth0_info
```

Available fields:
- `.Discovered.eth0_info.Exists` - boolean
- `.Discovered.eth0_info.Raw.addrs` - []string
- `.Discovered.eth0_info.Raw.mtu` - int

## Complete Example

Here's a complete example that demonstrates the discovery feature:

```yaml
# discovery.yaml

# Discovery phase - runs first
discovery:
  auditd_check:
    type: package
    resource:
      name: auditd
    register: auditd_installed

  docker_check:
    type: package
    resource:
      name: docker
    register: docker_installed

  app_version:
    type: command
    resource:
      command: "cat /etc/app/version"
    register: app_version

# Tests - run after discovery
# These tests only run if auditd is installed
{{ if .Discovered.auditd_installed.Installed }}
file:
  /etc/audit/auditd.conf:
    exists: true
    contains:
      - 'log_file = /var/log/audit/audit.log'
    meta:
      desc: "Auditd configuration file should exist and be properly configured"

service:
  auditd:
    enabled: true
    running: true
    meta:
      desc: "Auditd service should be enabled and running"
{{ end }}

# These tests only run if docker is installed
{{ if .Discovered.docker_installed.Installed }}
service:
  docker:
    enabled: true
    running: true

user:
  docker:
    exists: true
{{ end }}

# Version-specific tests
{{ if .Discovered.app_version.Value }}
{{ if regexMatch "^2\\..*" .Discovered.app_version.Value }}
file:
  /etc/app/v2-config.json:
    exists: true
    meta:
      desc: "Version 2 configuration should exist"
{{ else }}
file:
  /etc/app/v1-config.json:
    exists: true
    meta:
      desc: "Version 1 configuration should exist"
{{ end }}
{{ end }}
```

## Advanced Usage

### Multiple Conditions

You can combine multiple discovered values:

```yaml
{{ if and .Discovered.nginx_installed.Installed .Discovered.php_installed.Installed }}
file:
  /etc/nginx/conf.d/php.conf:
    exists: true
{{ end }}
```

### Version Comparisons

```yaml
{{ if .Discovered.package_version.Version }}
{{ if regexMatch "^2\\..*" .Discovered.package_version.Version }}
# Tests for version 2.x
{{ else if regexMatch "^1\\..*" .Discovered.package_version.Version }}
# Tests for version 1.x
{{ end }}
{{ end }}
```

### Accessing Raw Values

All discovered values have a `Raw` field that contains all the information:

```yaml
{{ if .Discovered.app_user.Raw.uid }}
{{ if eq .Discovered.app_user.Raw.uid 1000 }}
# Tests specific to UID 1000
{{ end }}
{{ end }}
```

## Skip Option

You can skip a discovery if needed:

```yaml
discovery:
  optional_check:
    type: package
    resource:
      name: optional-package
    register: optional
    skip: true
```

## Limitations

1. Discoveries are not supported when reading from STDIN (`goss validate -`)
2. Discoveries run once before template processing - they don't update dynamically
3. Discovery resources are not included in the test output

## Best Practices

1. **Use meaningful register names**: Choose descriptive names for your discovered values
2. **Check for nil values**: Always check if a value exists before using it
3. **Keep discoveries simple**: Discovery should be quick checks, not complex operations
4. **Document your discoveries**: Add comments explaining what each discovery does
5. **Group related discoveries**: Organize discoveries logically in your Goss file

## Template Functions

You can use Sprig template functions with discovered values:

```yaml
{{ if .Discovered.mycheck.Installed }}
{{ if eq .Discovered.mycheck.Version "1.2.3" }}
# Exact version match
{{ end }}
{{ end }}
```

Common functions:
- `eq`, `ne`, `lt`, `le`, `gt`, `ge` - Comparison
- `and`, `or`, `not` - Logical operators
- `regexMatch` - Regular expression matching
- See [Sprig documentation](http://masterminds.github.io/sprig/) for more functions

