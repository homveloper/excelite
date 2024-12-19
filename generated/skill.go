package models

import (
    "fmt"
    "gorm.io/gorm"
    "reflect"

)

type Skill struct {
    gorm.Model
    Cooldown float64 
    Effects []string 
    Element string 
    Index string 
    Mp_cost int32 
    Name string `gorm:"index"`
    Power float64 
    Requirements []string 
    Target_type string 

}


// BeforeSave handles array field serialization
func (m *Skill) BeforeSave(tx *gorm.DB) error {
    // Handle Effects array
    if m.Effects != nil {
        for i, v := range m.Effects {
            fieldName := fmt.Sprintf("Effects_%d", i)
            tx.Statement.SetColumn(fieldName, v)
        }
    }
    // Handle Requirements array
    if m.Requirements != nil {
        for i, v := range m.Requirements {
            fieldName := fmt.Sprintf("Requirements_%d", i)
            tx.Statement.SetColumn(fieldName, v)
        }
    }

    return nil
}

// AfterFind handles array field deserialization
func (m *Skill) AfterFind(tx *gorm.DB) error {
    // Initialize Effects array
    m.Effects = make([]string, 0)
    for i := 0; ; i++ {
        field := reflect.ValueOf(m).Elem().FieldByName(fmt.Sprintf("Effects_%d", i))
        if !field.IsValid() {
            break
        }
        if !field.IsZero() {
            m.Effects = append(m.Effects, field.Interface().(string))
        }
    }
    // Initialize Requirements array
    m.Requirements = make([]string, 0)
    for i := 0; ; i++ {
        field := reflect.ValueOf(m).Elem().FieldByName(fmt.Sprintf("Requirements_%d", i))
        if !field.IsValid() {
            break
        }
        if !field.IsZero() {
            m.Requirements = append(m.Requirements, field.Interface().(string))
        }
    }

    return nil
}
