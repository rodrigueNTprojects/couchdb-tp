// pkg/conflict/resolver.go - VERSION À COMPLÉTER
package conflict

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"couchdb-tp/pkg/client"
)

type ConflictResolver struct {
	client *client.Client
	dbName string
}

type ConflictDocument struct {
	ID         string                 `json:"_id"`
	Rev        string                 `json:"_rev"`
	Conflicts  []string               `json:"_conflicts,omitempty"`
	LedgerType string                 `json:"ledger_type"`
	AuditTrail map[string]interface{} `json:"audit_trail"`
}

func NewConflictResolver(client *client.Client, dbName string) *ConflictResolver {
	return &ConflictResolver{
		client: client,
		dbName: dbName,
	}
}

// ResolveByTimestamp résout un conflit en gardant la version avec le timestamp le plus récent
func (cr *ConflictResolver) ResolveByTimestamp(docID string) error {
	log.Printf("Résolution conflit pour %s dans %s", docID, cr.dbName)

	// 1. Récupérer le document principal avec conflits
	resp, err := cr.client.Get(fmt.Sprintf("/%s/%s?conflicts=true", cr.dbName, docID))
	if err != nil {
		return fmt.Errorf("erreur récupération document: %v", err)
	}

	var mainDoc ConflictDocument
	if err := json.Unmarshal(resp.Body, &mainDoc); err != nil {
		return fmt.Errorf("erreur parsing document: %v", err)
	}

	if len(mainDoc.Conflicts) == 0 {
		log.Printf("Aucun conflit détecté pour %s", docID)
		return nil
	}

	log.Printf("Conflits détectés: %v", mainDoc.Conflicts)

	// 2. Récupérer la première révision conflictuelle
	conflictRev := mainDoc.Conflicts[0]
	conflictResp, err := cr.client.Get(fmt.Sprintf("/%s/%s?rev=%s", cr.dbName, docID, conflictRev))
	if err != nil {
		return fmt.Errorf("erreur récupération révision conflictuelle: %v", err)
	}

	var conflictDoc ConflictDocument
	if err := json.Unmarshal(conflictResp.Body, &conflictDoc); err != nil {
		return fmt.Errorf("erreur parsing révision conflictuelle: %v", err)
	}

	// SOLUTION : Comparer les timestamps
	mainTimeStr, ok := mainDoc.AuditTrail["created_at"].(string)
	if !ok {
		return fmt.Errorf("timestamp manquant dans le document principal")
	}

	conflictTimeStr, ok := conflictDoc.AuditTrail["created_at"].(string)
	if !ok {
		return fmt.Errorf("timestamp manquant dans la révision conflictuelle")
	}

	mainTime, err := time.Parse(time.RFC3339, mainTimeStr)
	if err != nil {
		return fmt.Errorf("erreur parsing timestamp principal: %v", err)
	}

	conflictTime, err := time.Parse(time.RFC3339, conflictTimeStr)
	if err != nil {
		return fmt.Errorf("erreur parsing timestamp conflictuel: %v", err)
	}

	// TODO ÉQUIPE A et B: Comparer les timestamps et supprimer la révision la plus ancienne
	
	
	if // TODO - 1: compléter ici - condition de comparaison ici {
		// La révision conflictuelle est plus récente, supprimer la révision principale
		log.Printf("Révision conflictuelle plus récente (%s > %s) - supprimer révision principale",
			conflictTimeStr, mainTimeStr)
		_, err := // TODO - 2: compléter ici - appel à Delete() pour supprimer mainDoc.Rev
		if err != nil {
			return fmt.Errorf("erreur suppression révision principale: %v", err)
		}
	} else {
		// La révision principale est plus récente (ou égale), supprimer la révision conflictuelle
		log.Printf("Révision principale plus récente (%s >= %s) - supprimer révision %s",
			mainTimeStr, conflictTimeStr, conflictRev)
		_, err := // TODO ÉQUIPE - 3: appel à Delete() pour supprimer conflictRev
		if err != nil {
			return fmt.Errorf("erreur suppression révision conflictuelle: %v", err)
		}
	}
	// ========== ZONE À COMPLÉTER - FIN ==========

	log.Printf("Résolution par timestamp terminée pour %s", docID)
	return nil
}

// ListConflicts liste tous les documents ayant des conflits
func (cr *ConflictResolver) ListConflicts() ([]string, error) {
	// Récupérer la liste de tous les documents
	resp, err := cr.client.Get(fmt.Sprintf("/%s/_all_docs", cr.dbName))
	if err != nil {
		return nil, fmt.Errorf("erreur récupération liste documents: %v", err)
	}

	var result struct {
		Rows []struct {
			ID string `json:"id"`
		} `json:"rows"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("erreur parsing liste documents: %v", err)
	}

	var conflictDocs []string

	// Pour chaque document, vérifier s'il a des conflits
	log.Printf("Vérification de %d documents...", len(result.Rows))

	for _, row := range result.Rows {
		// Ignorer les documents de design
		if strings.HasPrefix(row.ID, "_design/") {
			continue
		}

		// Récupérer le document avec le paramètre conflicts=true
		docResp, err := cr.client.Get(fmt.Sprintf("/%s/%s?conflicts=true", cr.dbName, row.ID))
		if err != nil {
			log.Printf("Avertissement: impossible de lire %s: %v", row.ID, err)
			continue
		}

		var doc struct {
			Conflicts []string `json:"_conflicts"`
		}

		if err := json.Unmarshal(docResp.Body, &doc); err != nil {
			log.Printf("Avertissement: impossible de parser %s: %v", row.ID, err)
			continue
		}

		// Si le document a des conflits, l'ajouter à la liste
		if len(doc.Conflicts) > 0 {
			conflictDocs = append(conflictDocs, row.ID)
			log.Printf("Conflit détecté: %s (%d révisions conflictuelles)", row.ID, len(doc.Conflicts))
		}
	}

	return conflictDocs, nil
}
