package models

import (
    "fmt"
    "gorm.io/gorm"
    "reflect"

)

type Item struct {
    gorm.Model
    Effects []string 
    Index string 
    Level_req int32 
    Name string `gorm:"index"`
    Price int32 
    Rarity int32 
    Stackable bool 
    Type string 
    Weight float64 

}


// BeforeSave handles array field serialization
func (m *Item) BeforeSave(tx *gorm.DB) error {
    // Handle Effects array
    if m.Effects != nil {
        for i, v := range m.Effects {
            fieldName := fmt.Sprintf("Effects_%d", i)
            tx.Statement.SetColumn(fieldName, v)
        }
    }

    return nil
}

// AfterFind handles array field deserialization
func (m *Item) AfterFind(tx *gorm.DB) error {
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

    return nil
}
