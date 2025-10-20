// cmd/scenario_offline_equipA.go
// Scénario Équipe A : Panne pendant écriture (Nœud NA2)
// Programme Go cross-platform pour tester la résilience

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"couchdb-tp/pkg/client"
)

func main() {
	log.Println("=== SCENARIO EQUIPE A : PANNE NA2 PENDANT ECRITURE ===")

	// 1. Créer un client pour NA1
	na1Client, err := client.New("http://localhost:5987", "admin", "ecommerce2024")
	if err != nil {
		log.Fatalf("Erreur création client NA1: %v", err)
	}

	log.Println("Étape 0/6 : Nettoyage des documents existants...")
	
	// Supprimer le document s'il existe déjà
	deleteExistingDoc(na1Client, "/ecommerce_orders/test_order_conflict")
	
	// Attendre un peu pour la réplication de la suppression
	time.Sleep(3 * time.Second)

	log.Println("Étape 1/6 : Création du document initial sur NA1...")

	// Créer un document initial
	initialDoc := map[string]interface{}{
		"_id":        "test_order_conflict",
		"ledger_type": "commercial_transaction",
		"immutable": true,
		"transaction_hash": "initial_hash",
		"timestamp": time.Now().Format(time.RFC3339),
		"audit_trail": map[string]interface{}{
			"created_by": "scenario_a",
			"created_at": time.Now().Format(time.RFC3339),
			"source_node": "NA1",
			"validation_hash": "initial_hash",
			"integrity_check": "validated",
			"ledger_version": "1.0",
		},
		"transaction_data": map[string]interface{}{
			"order_id": "test_001",
			"amount":   100,
			"status":   "pending",
		},
	}

	resp, err := na1Client.Put("/ecommerce_orders/test_order_conflict", initialDoc)
	if err != nil {
		log.Fatalf("Erreur création document: %v", err)
	}

	// Récupérer le _rev du document créé
	var createResp map[string]interface{}
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		log.Fatalf("Erreur parsing réponse création: %v", err)
	}

	if errMsg, hasError := createResp["error"]; hasError {
		log.Fatalf("Erreur CouchDB lors de la création: %v - %v", errMsg, createResp["reason"])
	}

	originalRev := extractRevFromResponse(createResp, "rev")
	log.Printf("✓ Document créé - Status: %d, Rev: %s", resp.StatusCode, originalRev)

	// 2. Attendre la réplication
	log.Println("Étape 2/6 : Attente de la réplication (5 secondes)...")
	time.Sleep(5 * time.Second)

	// 3. Simuler une panne de NA2
	log.Println("Étape 3/6 : SIMULATION - Panne du nœud NA2...")
	if err := dockerCommand("stop", "couchdb-na-2"); err != nil {
		log.Printf("Avertissement arrêt NA2: %v", err)
	}
	time.Sleep(3 * time.Second)
	log.Println("✓ NA2 arrêté")

	// 4. Modifier le document sur NA1 pendant que NA2 est arrêté
	log.Println("Étape 4/6 : Modification sur NA1 pendant panne NA2...")

	getResp, err := na1Client.Get("/ecommerce_orders/test_order_conflict")
	if err != nil {
		log.Fatalf("Erreur récupération document: %v", err)
	}

	var currentDoc map[string]interface{}
	if err := json.Unmarshal(getResp.Body, &currentDoc); err != nil {
		log.Fatalf("Erreur parsing document actuel: %v", err)
	}

	currentRev := extractRev(currentDoc, "_rev")
	log.Printf("Révision actuelle avant modification: %s", currentRev)

	// Modifier avec le bon _rev
	modifiedDocNA1 := map[string]interface{}{
		"_id":  "test_order_conflict",
		"_rev": currentRev,
		"ledger_type": "commercial_transaction",
		"immutable": true,
		"transaction_hash": "hash_na1_modified",
		"timestamp": time.Now().Format(time.RFC3339),
		"audit_trail": map[string]interface{}{
			"created_by": "scenario_a",
			"created_at": time.Now().Format(time.RFC3339),
			"source_node": "NA1",
			"validation_hash": "hash_na1_modified",
			"integrity_check": "validated",
			"ledger_version": "1.0",
			"modification": "during_na2_outage",
		},
		"transaction_data": map[string]interface{}{
			"order_id":                "test_001",
			"amount":                  150,
			"status":                  "modified_na1",
			"modified_during_outage":  true,
		},
	}

	resp, err = na1Client.Put("/ecommerce_orders/test_order_conflict", modifiedDocNA1)
	if err != nil {
		log.Fatalf("Erreur modification document: %v", err)
	}
	
	var modifyResp map[string]interface{}
	json.Unmarshal(resp.Body, &modifyResp)
	na1NewRev := extractRevFromResponse(modifyResp, "rev")
	
	log.Printf("✓ Document modifié sur NA1 - Status: %d, Nouvelle Rev: %s", resp.StatusCode, na1NewRev)

	// 5. Redémarrer NA2 et créer le conflit
	log.Println("Étape 5/6 : Redémarrage NA2 et création FORCÉE du conflit...")
	if err := dockerCommand("start", "couchdb-na-2"); err != nil {
		log.Fatalf("Erreur redémarrage NA2: %v", err)
	}

	log.Println("Attente que NA2 soit opérationnel...")
	time.Sleep(10 * time.Second)

	// Créer un client pour NA2
	na2Client, err := client.New("http://localhost:5988", "admin", "ecommerce2024")
	if err != nil {
		log.Fatalf("Erreur création client NA2: %v", err)
	}

	// Créer une révision conflictuelle en utilisant _bulk_docs avec new_edits=false
	// Cela permet d'insérer une révision qui diverge de l'historique existant
	log.Println("Création d'une révision conflictuelle sur NA2 avec _bulk_docs...")
	
	conflictRev := "4-conflictna2" // Numéro de génération identique mais hash différent
	
	conflictDoc := map[string]interface{}{
		"_id":  "test_order_conflict",
		"_rev": conflictRev,
		"ledger_type": "commercial_transaction",
		"immutable": true,
		"transaction_hash": "hash_na2_conflict",
		"timestamp": time.Now().Format(time.RFC3339),
		"audit_trail": map[string]interface{}{
			"created_by": "scenario_a",
			"created_at": time.Now().Add(5 * time.Second).Format(time.RFC3339),
			"source_node": "NA2",
			"validation_hash": "hash_na2_conflict",
			"integrity_check": "validated",
			"ledger_version": "1.0",
			"modification": "na2_alternative_version",
		},
		"transaction_data": map[string]interface{}{
			"order_id": "test_001",
			"amount":   200,
			"status":   "modified_na2",
			"alternative_version": true,
		},
		"_revisions": map[string]interface{}{
			"start": 4,
			"ids": []string{"conflictna2", "thirdrev", "secondrev", originalRev[2:]},
		},
	}

	bulkPayload := map[string]interface{}{
		"docs": []interface{}{conflictDoc},
		"new_edits": false, // CRITIQUE: permet d'insérer des révisions arbitraires
	}

	resp, err = na2Client.Post("/ecommerce_orders/_bulk_docs", bulkPayload)
	if err != nil {
		log.Printf("Erreur création conflit via _bulk_docs: %v", err)
	} else {
		var bulkResp []interface{}
		json.Unmarshal(resp.Body, &bulkResp)
		log.Printf("✓ Conflit créé via _bulk_docs - Status: %d", resp.StatusCode)
		log.Printf("Réponse: %+v", bulkResp)
	}

	// 6. Attendre la synchronisation
	log.Println("Étape 6/6 : Attente synchronisation (15 secondes)...")
	time.Sleep(15 * time.Second)

	// Vérifier les conflits
	log.Println("\n=== VÉRIFICATION DES CONFLITS ===")
	
	// Vérifier sur NA1
	log.Println("\nVérification sur NA1:")
	checkConflicts(na1Client, "/ecommerce_orders/test_order_conflict", "NA1")
	
	// Vérifier sur NA2
	log.Println("\nVérification sur NA2:")
	checkConflicts(na2Client, "/ecommerce_orders/test_order_conflict", "NA2")

	log.Println("\n=== SCENARIO EQUIPE A TERMINE ===")
	log.Println("")
	log.Println("Instructions de test des vues:")
	log.Println("  1. Testez votre vue 'outage_impact_audit' pour voir les marqueurs de panne")
	log.Println("  2. Vérifiez 'node_sync_performance' pour analyser la synchronisation")
	log.Println("  3. Utilisez le resolver pour résoudre le conflit généré:")
	log.Println("     Windows: .\\bin\\conflict-resolver.exe -database ecommerce_orders -list")
	log.Println("     Linux/macOS: ./bin/conflict-resolver -database ecommerce_orders -list")
}

// checkConflicts vérifie et affiche les conflits sur un nœud
func checkConflicts(client *client.Client, path, nodeName string) {
	resp, err := client.Get(path + "?conflicts=true")
	if err != nil {
		log.Printf("❌ Erreur vérification %s: %v", nodeName, err)
		return
	}

	var doc map[string]interface{}
	json.Unmarshal(resp.Body, &doc)

	log.Printf("Status: %d", resp.StatusCode)
	log.Printf("Révision gagnante: %s", doc["_rev"])

	if conflicts, ok := doc["_conflicts"]; ok {
		conflictList := conflicts.([]interface{})
		log.Printf("✓✓✓ CONFLIT DÉTECTÉ: %d révision(s) conflictuelle(s) ✓✓✓", len(conflictList))
		log.Printf("Révisions en conflit: %v", conflicts)
		fmt.Printf("\nDocument complet:\n%s\n", prettyJSON(doc))
	} else {
		log.Println("✗ Aucun conflit détecté")
		fmt.Printf("Document actuel:\n%s\n", prettyJSON(doc))
	}
}

// prettyJSON formate un JSON de manière lisible
func prettyJSON(data interface{}) string {
	bytes, _ := json.MarshalIndent(data, "", "  ")
	return string(bytes)
}

// deleteExistingDoc supprime un document s'il existe
func deleteExistingDoc(client *client.Client, path string) {
	resp, err := client.Get(path)
	if err != nil {
		log.Printf("Document n'existe pas (normal au premier lancement)")
		return
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return
	}

	if revInterface, exists := doc["_rev"]; exists && revInterface != nil {
		if rev, ok := revInterface.(string); ok {
			deleteResp, err := client.Delete(path + "?rev=" + rev)
			if err != nil {
				log.Printf("Erreur suppression document: %v", err)
			} else {
				log.Printf("✓ Document existant supprimé (rev: %s, status: %d)", rev, deleteResp.StatusCode)
			}
		}
	}
}

// extractRev extrait de manière sécurisée le _rev d'un document
func extractRev(doc map[string]interface{}, fieldName string) string {
	if revInterface, exists := doc[fieldName]; exists && revInterface != nil {
		if revStr, ok := revInterface.(string); ok {
			return revStr
		}
		log.Fatalf("Le champ '%s' n'est pas une string: %v", fieldName, revInterface)
	}
	log.Fatalf("Le champ '%s' n'existe pas: %+v", fieldName, doc)
	return ""
}

// extractRevFromResponse extrait le rev d'une réponse CouchDB
func extractRevFromResponse(resp map[string]interface{}, fieldName string) string {
	if revInterface, exists := resp[fieldName]; exists && revInterface != nil {
		if revStr, ok := revInterface.(string); ok {
			return revStr
		}
		log.Fatalf("Le champ '%s' n'est pas une string: %v", fieldName, revInterface)
	}
	log.Fatalf("Le champ '%s' n'existe pas dans la réponse: %+v", fieldName, resp)
	return ""
}

// dockerCommand exécute une commande Docker de manière cross-platform
func dockerCommand(action, container string) error {
	cmd := exec.Command("docker", action, container)
	return cmd.Run()
}