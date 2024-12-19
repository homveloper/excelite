package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

const (
	SheetCharacter = "Character"
	SheetItem      = "Item"
	SheetSkill     = "Skill"
	SheetRelation  = "Relation"
)

func main() {
	log.Println("Starting Excel file generation...")

	// 현재 작업 디렉토리 확인
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}
	log.Printf("Current working directory: %s", currentDir)

	// 새 Excel 파일 생성
	f := excelize.NewFile()
	log.Println("Created new Excel file")

	// Character 시트 생성
	log.Println("Generating Character sheet...")
	if err := generateCharacterSheet(f); err != nil {
		log.Fatal("Failed to generate Character sheet:", err)
	}
	log.Printf("Character sheet created. Total sheets: %v", f.GetSheetList())

	// Item 시트 생성
	log.Println("Generating Item sheet...")
	if err := generateItemSheet(f); err != nil {
		log.Fatal("Failed to generate Item sheet:", err)
	}
	log.Printf("Item sheet created. Total sheets: %v", f.GetSheetList())

	// Skill 시트 생성
	log.Println("Generating Skill sheet...")
	if err := generateSkillSheet(f); err != nil {
		log.Fatal("Failed to generate Skill sheet:", err)
	}
	log.Printf("Skill sheet created. Total sheets: %v", f.GetSheetList())

	// // Relation 시트 생성
	// log.Println("Generating Relation sheet...")
	// if err := generateRelationSheet(f); err != nil {
	// 	log.Fatal("Failed to generate Relation sheet:", err)
	// }
	// log.Printf("Relation sheet created. Total sheets: %v", f.GetSheetList())

	// Sheet1 삭제 시도
	if _, err := f.GetSheetIndex("Sheet1"); err == nil {
		f.DeleteSheet("Sheet1")
		log.Println("Deleted default Sheet1")
	} else {
		log.Fatalf("Failed to delete Sheet1: %v", err)
	}

	// 저장할 파일 경로 설정
	filename := "game_data.xlsx"
	filepath := filepath.Join(currentDir, filename)
	log.Printf("Saving Excel file to: %s", filepath)

	// 파일 저장
	if err := f.SaveAs(filepath); err != nil {
		log.Fatal("Failed to save Excel file:", err)
	}

	// 파일이 실제로 생성되었는지 확인
	if _, err := os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			log.Fatal("File was not created:", err)
		}
		log.Fatal("Error checking file:", err)
	}

	log.Printf("Excel file successfully generated at: %s", filepath)

	// 생성된 파일의 크기 확인
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		log.Fatal("Error getting file info:", err)
	}
	log.Printf("File size: %d bytes", fileInfo.Size())
}

// writeSheet 함수에도 로깅 추가
func writeSheet(f *excelize.File, sheetName string, headers []string, tags []string, types []string, data [][]interface{}) error {
	// 새 시트 생성
	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %v", sheetName, err)
	}
	f.SetActiveSheet(idx)

	// 헤더 (1행)
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to write header %s: %v", header, err)
		}
	}

	// 태그 (2행)
	for i, tag := range tags {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		if err := f.SetCellValue(sheetName, cell, tag); err != nil {
			return fmt.Errorf("failed to write tag %s: %v", tag, err)
		}
	}

	// 타입 (3행)
	for i, typ := range types {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		if err := f.SetCellValue(sheetName, cell, typ); err != nil {
			return fmt.Errorf("failed to write type %s: %v", typ, err)
		}
	}

	// 데이터 (4행부터)
	for rowNum, row := range data {
		for colNum, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colNum+1, rowNum+4)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("failed to write data at %s: %v", cell, err)
			}
		}
	}

	return nil
}

func generateCharacterSheet(f *excelize.File) error {
	headers := []string{"index", "name", "type", "level", "hp", "mp", "attack", "defense", "speed", "class",
		"skills", "skills", "skills"}
	tags := []string{"all", "all,index", "all", "all", "all", "all", "all", "all", "all", "all",
		"all", "all", "all"}
	types := []string{"string", "string", "string", "int32", "float64", "float64", "float64", "float64", "float64", "string",
		"array<string>", "array<string>", "array<string>"}
	data := [][]interface{}{
		{1, "Warrior", "player", 1, 100.0, 50.0, 10.0, 8.0, 5.0, "fighter", "slash", "bash", "defend"},
		{2, "Mage", "player", 1, 70.0, 120.0, 5.0, 4.0, 4.0, "wizard", "fireball", "ice_bolt", "teleport"},
		{3, "Rogue", "player", 1, 80.0, 60.0, 8.0, 5.0, 9.0, "thief", "backstab", "dodge", "stealth"},
	}

	return writeSheet(f, SheetCharacter, headers, tags, types, data)
}

func generateItemSheet(f *excelize.File) error {
	headers := []string{"index", "name", "type", "rarity", "level_req", "price", "weight", "description",
		"effects", "effects", "stackable"}
	tags := []string{"all", "all,index", "all", "all", "all", "all", "", "design",
		"all", "all", "all"}
	types := []string{"string", "string", "string", "int32", "int32", "int32", "float64", "string",
		"array<string>", "array<string>", "bool"}
	data := [][]interface{}{
		{1, "Iron Sword", "weapon", 1, 1, 100, 2.5, "Basic sword", "damage+5", "durability:100", false},
		{2, "Health Potion", "consumable", 1, 1, 50, 0.3, "Restores HP", "heal:50", "duration:instant", true},
		{3, "Magic Staff", "weapon", 2, 5, 500, 1.8, "Magic staff", "magic+10", "mana+20", false},
	}

	return writeSheet(f, SheetItem, headers, tags, types, data)
}

func generateSkillSheet(f *excelize.File) error {
	headers := []string{"index", "name", "element", "power", "mp_cost", "cooldown", "target_type",
		"effects", "effects", "requirements", "requirements"}
	tags := []string{"all", "all,index", "all", "all", "all", "all", "all",
		"all", "all", "all", "all"}
	types := []string{"string", "string", "string", "float64", "int32", "float64", "string",
		"array<string>", "array<string>", "array<string>", "array<string>"}
	data := [][]interface{}{
		{1, "Slash", "physical", 20.0, 0, 1.5, "single", "bleeding", "stun", "sword", "level:1"},
		{2, "Fireball", "fire", 35.0, 25, 3.0, "area", "burn", "knockback", "staff", "level:3"},
		{3, "Stealth", "neutral", 0.0, 30, 15.0, "self", "invisibility", "move_speed+", "dagger", "level:2"},
	}

	return writeSheet(f, SheetSkill, headers, tags, types, data)
}

func generateRelationSheet(f *excelize.File) error {
	headers := []string{"SourceTable", "TargetTable", "RelationType", "ForeignKey", "ReferenceKey"}
	data := [][]string{
		{"Character", "Skill", "hasMany", "CharacterID", "ID"},
		{"Character", "Item", "hasMany", "CharacterID", "ID"},
	}

	// 시트 생성
	index, err := f.NewSheet(SheetRelation)
	if err != nil {
		return err
	}

	f.SetActiveSheet(index)

	// 헤더 작성
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(SheetRelation, cell, header); err != nil {
			return err
		}
	}

	// 데이터 작성
	for i, row := range data {
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			if err := f.SetCellValue(SheetRelation, cell, value); err != nil {
				return err
			}
		}
	}

	return nil
}
