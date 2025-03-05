package addCustomer

import (
	"encoding/json"
	"errors"
	"***REMOVED***/taskflow"
	"io"
	"net/http"
)

type Input struct {
	UserId   uint   `json:"user_id"`
	Provider string `json:"provider"`
}

type AddCustomer struct {
	payload Input
	output  string
}

func New(payload *string) (taskflow.Job, error) {
	var customer AddCustomer
	err := json.Unmarshal([]byte(*payload), &customer.payload)
	if err != nil {
		return &AddCustomer{}, errors.New("invalid payload")
	}
	customer.output = ""
	return &customer, nil
}

func (a *AddCustomer) Run() error {
	resp, err := http.Get("https://webhook.site/4ae8639d-3464-4683-a164-0c93cf245cfd")
	if err != nil {
		a.output = err.Error()
		return err
	}

	if resp != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			a.output = err.Error()
			return err
		}

		a.output = string(body)

		resp.Body.Close()
	} else {
		return errors.New("empty response")
	}

	return nil
}

func (a *AddCustomer) GetOutput() *string {
	if a.output == "" {
		return nil
	}
	return &a.output
}
