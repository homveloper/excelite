package models

import (
    "fmt"
    "gorm.io/gorm"
    "reflect"

)

type Character struct {
    gorm.Model
    Attack float64 
    Class string 
    Defense float64 
    Hp float64 
    Index string 
    Level int32 
    Mp float64 
    Name string `gorm:"index"`
    Skills []string 
    Speed float64 
    Type string 

}


// BeforeSave handles array field serialization
func (m *Character) BeforeSave(tx *gorm.DB) error {
    // Handle Skills array
    if m.Skills != nil {
        for i, v := range m.Skills {
            fieldName := fmt.Sprintf("Skills_%d", i)
            tx.Statement.SetColumn(fieldName, v)
        }
    }

    return nil
}

// AfterFind handles array field deserialization
func (m *Character) AfterFind(tx *gorm.DB) error {
    // Initialize Skills array
    m.Skills = make([]string, 0)
    for i := 0; ; i++ {
        field := reflect.ValueOf(m).Elem().FieldByName(fmt.Sprintf("Skills_%d", i))
        if !field.IsValid() {
            break
        }
        if !field.IsZero() {
            m.Skills = append(m.Skills, field.Interface().(string))
        }
    }

    return nil
}
