package apiv1

import (
	"context"
	"errors"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	hreq "github.com/imroc/req/v3"
	"github.com/superfly/flyctl/api"
	"net/http"
	"time"
)

var NonceHeader = "fly-machine-lease-nonce"

type MachineAPI struct {
	client     *graphql.Client
	httpClient *hreq.Client
	endpoint   string
}

func NewMachineAPI(httpClient *hreq.Client, endpoint string) *MachineAPI {
	return &MachineAPI{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func (a *MachineAPI) LockMachine(app string, id string, timeout int) (*api.MachineLease, error) {
	var res api.MachineLease
	_, err := a.httpClient.R().SetSuccessResult(&res).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/lease/?ttl=%d", a.endpoint, app, id, timeout))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *MachineAPI) ReleaseMachine(lease api.MachineLease, app string, id string) error {
	_, err := a.httpClient.R().SetHeader(NonceHeader, lease.Data.Nonce).Delete(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/lease", a.endpoint, app, id))
	if err != nil {
		return err
	}
	return nil
}

func (a *MachineAPI) WaitForMachine(app string, id string, instanceID string) error {
	_, err := a.httpClient.R().Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/wait?instance_id=%s", a.endpoint, app, id, instanceID))
	return err
}

// CreateMachine takes a MachineCreateOrUpdateRequest and creates the requested machine in the given app and then writes the response into the `res` param
func (a *MachineAPI) CreateMachine(req api.Machine, app string) (*api.Machine, error) {
	var res api.Machine
	if req.Config != nil && req.Config.Guest != nil {
		if req.Config.Guest.CPUKind == "" {
			req.Config.Guest.CPUKind = "shared"
		}
		if req.Config.Guest.CPUs == 0 {
			req.Config.Guest.CPUs = 1
		}
		if req.Config.Guest.MemoryMB == 0 {
			req.Config.Guest.MemoryMB = 256
		}
	}
	createResponse, err := a.httpClient.R().SetBody(req).SetSuccessResult(res).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines", a.endpoint, app))

	if err != nil {
		return nil, err
	}

	if createResponse.StatusCode != http.StatusCreated && createResponse.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Create request failed: %s, %+v", createResponse.Status, createResponse))
	}
	return &res, nil
}

func (a *MachineAPI) UpdateMachine(req api.Machine, app string, id string) (*api.Machine, error) {
	var res api.Machine
	if req.Config != nil && req.Config.Guest != nil {
		if req.Config.Guest.CPUKind == "" {
			req.Config.Guest.CPUKind = "shared"
		}
		if req.Config.Guest.CPUs == 0 {
			req.Config.Guest.CPUs = 1
		}
		if req.Config.Guest.MemoryMB == 0 {
			req.Config.Guest.MemoryMB = 256
		}
	}
	lease, err := a.LockMachine(app, id, 30)
	if err != nil {
		return nil, err
	}
	reqRes, err := a.httpClient.R().SetBody(req).SetSuccessResult(&res).SetHeader(NonceHeader, lease.Data.Nonce).Post(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
	if err != nil {
		return nil, err
	}
	err = a.ReleaseMachine(*lease, app, id)
	if err != nil {
		return nil, err
	}
	if reqRes.StatusCode != http.StatusCreated && reqRes.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Update request failed: %s, %+v", reqRes.Status, reqRes))
	}
	return &res, nil
}

func (a *MachineAPI) ReadMachine(app string, id string) (*api.Machine, error) {
	var res api.Machine
	_, err := a.httpClient.R().SetSuccessResult(&res).Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *MachineAPI) DeleteMachine(app string, id string, maxRetries int) error {
	deleted := false
	for i := 0; i < maxRetries; i++ {
		var machine *api.Machine
		readResponse, err := a.httpClient.R().SetSuccessResult(&machine).Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s", a.endpoint, app, id))
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

func (a *MachineAPI) CreateVolume(ctx context.Context, name, app, region string, size int) (*api.Volume, error) {
	var res api.Volume
	_, err := a.httpClient.R().SetContext(ctx).SetBody(api.CreateVolumeRequest{
		Name:   name,
		Region: region,
		SizeGb: &size,
	}).SetSuccessResult(&res).Post(fmt.Sprintf("http://%s/v1/apps/%s/volumes", a.endpoint, app))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (a *MachineAPI) GetVolume(ctx context.Context, id, app string) (*api.Volume, error) {
	var res api.Volume
	_, err := a.httpClient.R().SetContext(ctx).SetSuccessResult(&res).Get(fmt.Sprintf("http://%s/v1/apps/%s/volumes/%s", a.endpoint, app, id))
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (a *MachineAPI) DeleteVolume(ctx context.Context, id, app string) error {
	_, err := a.httpClient.R().SetContext(ctx).Delete(fmt.Sprintf("http://%s/v1/apps/%s/volumes/%s", a.endpoint, app, id))
	if err != nil {
		return err
	}
	return nil
}
