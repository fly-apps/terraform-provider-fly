package apiv1

import (
	"errors"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	hreq "github.com/imroc/req/v3"
	"time"
)

var NonceHeader = "fly-machine-lease-nonce"

type MachineAPI struct {
	client     *graphql.Client
	httpClient *hreq.Client
	endpoint   string
}

type MachineMount struct {
	Encrypted bool   `json:"encrypted,omitempty"`
	Path      string `json:"path"`
	SizeGb    int    `json:"size_gb,omitempty"`
	Volume    string `json:"volume"`
}

type Port struct {
	Port     int64    `json:"port"`
	Handlers []string `json:"handlers"`
}

type Service struct {
	Ports        []Port `json:"ports"`
	Protocol     string `json:"protocol"`
	InternalPort int64  `json:"internal_port"`
}

type InitConfig struct {
	Cmd        []string `json:"cmd,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
}

type MachineConfig struct {
	Image    string            `json:"image"`
	Env      map[string]string `json:"env"`
	Init     InitConfig        `json:"init,omitempty"`
	Mounts   []MachineMount    `json:"mounts,omitempty"`
	Services []Service         `json:"services"`
}

type GuestConfig struct {
	Cpus     int    `json:"cpus,omitempty"`
	MemoryMb int    `json:"memory_mb,omitempty"`
	CpuType  string `json:"cpu_type,omitempty"`
}

type MachineCreateOrUpdateRequest struct {
	Name   string        `json:"name"`
	Region string        `json:"region"`
	Config MachineConfig `json:"config"`
	Guest  GuestConfig   `json:"guest,omitempty"`
}

type MachineResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Region     string `json:"region"`
	InstanceID string `json:"instance_id"`
	PrivateIP  string `json:"private_ip"`
	Config     struct {
		Env  map[string]string `json:"env"`
		Init struct {
			//Exec       interface{} `json:"exec"`
			Entrypoint []string `json:"entrypoint"`
			Cmd        []string `json:"cmd"`
			//Tty        bool        `json:"tty"`
		} `json:"init"`
		Image    string      `json:"image"`
		Metadata interface{} `json:"metadata"`
		Restart  struct {
			Policy string `json:"policy"`
		} `json:"restart"`
		Services []Service      `json:"services"`
		Mounts   []MachineMount `json:"mounts"`
		Guest    struct {
			CPUKind  string `json:"cpu_kind"`
			Cpus     int    `json:"cpus"`
			MemoryMb int    `json:"memory_mb"`
		} `json:"guest"`
	} `json:"config"`
	ImageRef struct {
		Registry   string `json:"registry"`
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Digest     string `json:"digest"`
		Labels     struct {
		} `json:"labels"`
	} `json:"image_ref"`
	CreatedAt time.Time `json:"created_at"`
}

type MachineLease struct {
	Status string `json:"status"`
	Data   struct {
		Nonce     string `json:"nonce"`
		ExpiresAt int64  `json:"expires_at"`
		Owner     string `json:"owner"`
	}
}

func NewMachineAPI(httpClient *hreq.Client, endpoint string) *MachineAPI {
	return &MachineAPI{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func (a *MachineAPI) LockMachine(app string, id string, timeout int) (*MachineLease, error) {
	var res MachineLease
	_, err := a.httpClient.R().SetResult(&res).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/lease/?ttl=%d", a.endpoint, app, id, timeout))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *MachineAPI) ReleaseMachine(lease MachineLease, app string, id string) error {
	_, err := a.httpClient.R().SetHeader(NonceHeader, lease.Data.Nonce).Delete(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/lease", a.endpoint, app, id))
	if err != nil {
		return err
	}
	return nil
}

// CreateMachine takes a MachineCreateOrUpdateRequest and creates the requested machine in the given app and then writes the response into the `res` param
func (a *MachineAPI) CreateMachine(req MachineCreateOrUpdateRequest, app string, res *MachineResponse) (*hreq.Response, error) {
	return a.httpClient.R().SetBody(req).SetResult(res).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines", a.endpoint, app))
}

func (a *MachineAPI) UpdateMachine(req MachineCreateOrUpdateRequest, app string, id string, res *MachineResponse) (*hreq.Response, error) {
	lease, err := a.LockMachine(app, id, 30)
	if err != nil {
		return nil, err
	}
	reqRes, err := a.httpClient.R().SetBody(req).SetResult(res).SetHeader(NonceHeader, lease.Data.Nonce).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
	if err != nil {
		return nil, err
	}
	err = a.ReleaseMachine(*lease, app, id)
	if err != nil {
		return nil, err
	}
	return reqRes, nil
}

func (a *MachineAPI) ReadMachine(app string, id string, res *MachineResponse) (*hreq.Response, error) {
	return a.httpClient.R().SetResult(res).Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
}

func (a *MachineAPI) DeleteMachine(app string, id string, maxRetries int) error {
	deleted := false
	for i := 0; i < maxRetries; i++ {
		var machine MachineResponse
		readResponse, err := a.httpClient.R().SetResult(&machine).Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
		if err != nil {
			return err
		}

		if readResponse.StatusCode == 200 {
			if machine.State == "started" || machine.State == "starting" || machine.State == "replacing" {
				_, _ = a.httpClient.R().Post(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/stop", a.endpoint, app, id))
			}
			if machine.State == "stopping" || machine.State == "destroying" {
				time.Sleep(5 * time.Second)
			}
			if machine.State == "stopped" || machine.State == "replaced" {
				_, err = a.httpClient.R().Delete(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
				if err != nil {
					return err
				}
			}
			if machine.State == "destroyed" {
				deleted = true
				break
			}
		}
	}
	if !deleted {
		return errors.New("max retries exceeded")
	}
	return nil
}
