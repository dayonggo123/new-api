package ratio_setting

import (
	"encoding/json"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

// 模型级分组倍率覆盖：模型名 -> 分组名 -> 倍率
var defaultModelGroupRatio = map[string]map[string]float64{}

var modelGroupRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupGroupRatio = map[string]map[string]float64{
	"vip": {
		"edit_this": 0.9,
	},
}

var groupGroupRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupSpecialUsableGroup = map[string]map[string]string{
	"vip": {
		"append_1":   "vip_special_group_1",
		"-:remove_1": "vip_removed_group_1",
	},
}

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]            `json:"group_ratio"`
	ModelGroupRatio         *types.RWMap[string, map[string]float64] `json:"model_group_ratio"`
	GroupGroupRatio         *types.RWMap[string, map[string]float64] `json:"group_group_ratio"`
	GroupSpecialUsableGroup *types.RWMap[string, map[string]string]  `json:"group_special_usable_group"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupSpecialUsableGroup := types.NewRWMap[string, map[string]string]()
	groupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)

	groupRatioMap.AddAll(defaultGroupRatio)
	modelGroupRatioMap.AddAll(defaultModelGroupRatio)
	groupGroupRatioMap.AddAll(defaultGroupGroupRatio)

	groupRatioSetting = GroupRatioSetting{
		GroupSpecialUsableGroup: groupSpecialUsableGroup,
		GroupRatio:              groupRatioMap,
		ModelGroupRatio:         modelGroupRatioMap,
		GroupGroupRatio:         groupGroupRatioMap,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioSetting() *GroupRatioSetting {
	if groupRatioSetting.GroupSpecialUsableGroup == nil {
		groupRatioSetting.GroupSpecialUsableGroup = types.NewRWMap[string, map[string]string]()
		groupRatioSetting.GroupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	}
	if groupRatioSetting.ModelGroupRatio == nil {
		groupRatioSetting.ModelGroupRatio = types.NewRWMap[string, map[string]float64]()
		groupRatioSetting.ModelGroupRatio.AddAll(defaultModelGroupRatio)
	}
	return &groupRatioSetting
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupRatioMap, jsonStr)
}

func GetGroupRatio(name string) float64 {
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

// GetModelGroupRatio 获取模型级分组倍率覆盖，如果不存在则回退到全局 GroupRatio
func GetModelGroupRatio(modelName, groupName string) float64 {
	if modelGroupRatioMap == nil {
		return GetGroupRatio(groupName)
	}
	groupRatios, ok := modelGroupRatioMap.Get(modelName)
	if !ok || groupRatios == nil {
		return GetGroupRatio(groupName)
	}
	ratio, ok := groupRatios[groupName]
	if !ok {
		return GetGroupRatio(groupName)
	}
	return ratio
}

func ModelGroupRatio2JSONString() string {
	if modelGroupRatioMap == nil {
		return "{}"
	}
	return modelGroupRatioMap.MarshalJSONString()
}

func UpdateModelGroupRatioByJSONString(jsonStr string) error {
	if modelGroupRatioMap == nil {
		modelGroupRatioMap = types.NewRWMap[string, map[string]float64]()
	}
	return types.LoadFromJsonString(modelGroupRatioMap, jsonStr)
}

func GetModelGroupRatioCopy() map[string]map[string]float64 {
	if modelGroupRatioMap == nil {
		return map[string]map[string]float64{}
	}
	return modelGroupRatioMap.ReadAll()
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	gp, ok := groupGroupRatioMap.Get(userGroup)
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GroupGroupRatio2JSONString() string {
	return groupGroupRatioMap.MarshalJSONString()
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupGroupRatioMap, jsonStr)
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}

func CheckModelGroupRatio(jsonStr string) error {
	checkModelGroupRatio := make(map[string]map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &checkModelGroupRatio)
	if err != nil {
		return err
	}
	for modelName, groupRatios := range checkModelGroupRatio {
		for groupName, ratio := range groupRatios {
			if ratio < 0 {
				return errors.New("model group ratio must be not less than 0: " + modelName + "/" + groupName)
			}
		}
	}
	return nil
}
