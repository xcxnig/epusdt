package data

import (
	"math/rand"
	"strings"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/dromara/carbon/v2"
)

// ListRpcNodes returns rows optionally filtered by network.
func ListRpcNodes(network string) ([]mdb.RpcNode, error) {
	var rows []mdb.RpcNode
	tx := dao.Mdb.Model(&mdb.RpcNode{})
	if network != "" {
		tx = tx.Where("network = ?", strings.ToLower(network))
	}
	err := tx.Order("id ASC").Find(&rows).Error
	return rows, err
}

// GetRpcNodeByID fetches one row.
func GetRpcNodeByID(id uint64) (*mdb.RpcNode, error) {
	row := new(mdb.RpcNode)
	err := dao.Mdb.Model(row).Limit(1).Find(row, id).Error
	return row, err
}

// CreateRpcNode inserts a row.
func CreateRpcNode(row *mdb.RpcNode) error {
	return dao.Mdb.Create(row).Error
}

// UpdateRpcNodeFields patches mutable columns.
func UpdateRpcNodeFields(id uint64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return dao.Mdb.Model(&mdb.RpcNode{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteRpcNodeByID soft-deletes the row.
func DeleteRpcNodeByID(id uint64) error {
	return dao.Mdb.Where("id = ?", id).Delete(&mdb.RpcNode{}).Error
}

// SelectRpcNode picks a healthy RPC endpoint for a (network, type) pair.
// Strategy: weighted random among rows where enabled=true AND status=ok.
// Falls back to enabled rows with status=unknown when no health check has
// run yet. Explicitly down rows are not selected.
func SelectRpcNode(network, nodeType string) (*mdb.RpcNode, error) {
	var rows []mdb.RpcNode
	err := dao.Mdb.Model(&mdb.RpcNode{}).
		Where("network = ?", strings.ToLower(network)).
		Where("type = ?", strings.ToLower(nodeType)).
		Where("enabled = ?", true).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	healthy := make([]mdb.RpcNode, 0, len(rows))
	bootstrap := make([]mdb.RpcNode, 0, len(rows))
	for _, r := range rows {
		switch r.Status {
		case mdb.RpcNodeStatusOk:
			healthy = append(healthy, r)
		case "", mdb.RpcNodeStatusUnknown:
			bootstrap = append(bootstrap, r)
		}
	}
	candidates := healthy
	if len(candidates) == 0 {
		candidates = bootstrap
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	return pickWeighted(candidates), nil
}

// pickWeighted chooses one row using the Weight column. Weights < 1 are
// coerced to 1 so an admin-zeroed row still has a chance (use Enabled=false
// to truly disable).
func pickWeighted(rows []mdb.RpcNode) *mdb.RpcNode {
	total := 0
	for i := range rows {
		w := rows[i].Weight
		if w < 1 {
			w = 1
		}
		total += w
	}
	if total <= 0 {
		return &rows[0]
	}
	pick := rand.Intn(total)
	acc := 0
	for i := range rows {
		w := rows[i].Weight
		if w < 1 {
			w = 1
		}
		acc += w
		if pick < acc {
			return &rows[i]
		}
	}
	return &rows[len(rows)-1]
}

// UpdateRpcNodeHealth stamps status/latency + last_checked_at.
func UpdateRpcNodeHealth(id uint64, status string, latencyMs int) error {
	return dao.Mdb.Model(&mdb.RpcNode{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          status,
			"last_latency_ms": latencyMs,
			"last_checked_at": carbon.Now().StdTime(),
		}).Error
}
