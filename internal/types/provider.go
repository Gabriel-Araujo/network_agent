package types

import "errors"

type Provider struct {
	Name         string  `json:"name"`
	Url          string  `json:"url"`
	ApiKey       string  `json:"api_key"`
	DefaultModel int8    `json:"default_model"`
	Models       []Model `json:"models"`
}

func (p *Provider) AddModel(model Model) {
	p.Models = append(p.Models, model)
}

func (p *Provider) RemoveModel(modelId string) (Model, error) {
	for i, m := range p.Models {
		if m.Id == modelId {
			p.Models = append(p.Models[:i], p.Models[i+1:]...)
			return m, nil
		}
	}
	return Model{}, errors.New("model not found")
}
