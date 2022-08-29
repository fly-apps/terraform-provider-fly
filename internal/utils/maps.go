package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func KVToTfMap(kv map[string]string, elemType attr.Type) types.Map {
	var TFMap types.Map
	TFMap.ElemType = elemType
	for key, value := range kv {
		if TFMap.Elems == nil {
			TFMap.Elems = map[string]attr.Value{}
		}
		TFMap.Elems[key] = types.String{Value: value}
	}
	return TFMap
}
