package role

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-auth/authorization/resource"
	"github.com/fabric8-services/fabric8-auth/errors"
	"github.com/fabric8-services/fabric8-auth/gormsupport"
	"github.com/fabric8-services/fabric8-auth/log"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"

	"fmt"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type Role struct {
	gormsupport.Lifecycle

	// This is the primary key value
	RoleID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" gorm:"column:role_id"`
	// The resource type that this role applies to
	ResourceType resource.ResourceType `gorm:"ForeignKey:ResourceTypeID;AssociationForeignKey:ResourceTypeID"`
	// The foreign key value for ResourceType
	ResourceTypeID uuid.UUID
	// The name of this role
	Name string
}

// The scopes associated with this role
//Scopes []resource.ResourceTypeScope `gorm:"many2many:role_scope;AssociationForeignKey:resourceTypeScopeID;ForeignKey:roleID"`

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Role) TableName() string {
	return "role"
}

// GetLastModified returns the last modification time
func (m Role) GetLastModified() time.Time {
	return m.UpdatedAt
}

type RoleScope struct {
	gormsupport.Lifecycle

	RoleID uuid.UUID `sql:"type:uuid" gorm:"primary_key" gorm:"column:role_ID"`

	Scope   resource.ResourceTypeScope `gorm:"ForeignKey:ScopeID;AssociationForeignKey:ResourceTypeScopeID"`
	ScopeID uuid.UUID                  `sql:"type:uuid" gorm:"primary_key" gorm:"column:role_ID"`
}

func (m RoleScope) TableName() string {
	return "role_scope"
}

// GetLastModified returns the last modification time
func (m RoleScope) GetLastModified() time.Time {
	return m.UpdatedAt
}

// GormRoleRepository is the implementation of the storage interface for Role.
type GormRoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository creates a new storage type.
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &GormRoleRepository{db: db}
}

// RoleRepository represents the storage interface.
type RoleRepository interface {
	CheckExists(ctx context.Context, id string) (bool, error)
	Load(ctx context.Context, ID uuid.UUID) (*Role, error)
	Create(ctx context.Context, u *Role) error
	Save(ctx context.Context, u *Role) error
	List(ctx context.Context) ([]Role, error)
	Delete(ctx context.Context, ID uuid.UUID) error

	Lookup(ctx context.Context, name string, resourceType string) (*Role, error)
	ListScopes(ctx context.Context, u *Role) ([]resource.ResourceTypeScope, error)
	AddScope(ctx context.Context, u *Role, s *resource.ResourceTypeScope) error
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormRoleRepository) TableName() string {
	return "role"
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (m *GormRoleRepository) CheckExists(ctx context.Context, id string) (bool, error) {
	defer goa.MeasureSince([]string{"goa", "db", "role", "exists"}, time.Now())

	var exists bool
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				role_id=$1
				AND deleted_at IS NULL
		)`, m.TableName())

	err := m.db.CommonDB().QueryRow(query, id).Scan(&exists)
	if err == nil && !exists {
		return exists, errors.NewNotFoundError(m.TableName(), id)
	}
	if err != nil {
		return false, errors.NewInternalError(ctx, errs.Wrapf(err, "unable to verify if %s exists", m.TableName()))
	}
	return exists, nil
}

// CRUD Functions

// Load returns a single Role as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *GormRoleRepository) Load(ctx context.Context, id uuid.UUID) (*Role, error) {
	defer goa.MeasureSince([]string{"goa", "db", "role", "load"}, time.Now())
	var native Role
	err := m.db.Table(m.TableName()).Preload("ResourceType"). /*.Preload("Scopes")*/ Where("role_id = ?", id).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundError("role", id.String())
	}
	return &native, errs.WithStack(err)
}

// Create creates a new record.
func (m *GormRoleRepository) Create(ctx context.Context, u *Role) error {
	defer goa.MeasureSince([]string{"goa", "db", "role", "create"}, time.Now())
	if u.RoleID == uuid.Nil {
		u.RoleID = uuid.NewV4()
	}
	err := m.db.Create(u).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"role_id": u.RoleID,
			"err":     err,
		}, "unable to create the role")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"role_id": u.RoleID,
	}, "Role created!")
	return nil
}

// Save modifies a single record
func (m *GormRoleRepository) Save(ctx context.Context, model *Role) error {
	defer goa.MeasureSince([]string{"goa", "db", "role", "save"}, time.Now())

	obj, err := m.Load(ctx, model.RoleID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"role_id": model.RoleID,
			"err":     err,
		}, "unable to update role")
		return errs.WithStack(err)
	}
	err = m.db.Model(obj).Updates(model).Error
	if err != nil {
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"role_id": model.RoleID,
	}, "Role saved!")
	return nil
}

// Delete removes a single record.
func (m *GormRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "role", "delete"}, time.Now())

	obj := Role{RoleID: id}

	err := m.db.Delete(&obj).Error

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"role_id": id,
			"err":     err,
		}, "unable to delete the role")
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"role_id": id,
	}, "Role deleted!")

	return nil
}

// List returns all roles
func (m *GormRoleRepository) List(ctx context.Context) ([]Role, error) {
	defer goa.MeasureSince([]string{"goa", "db", "role", "list"}, time.Now())
	var rows []Role

	err := m.db.Model(&Role{}).Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}
	return rows, nil
}

func (m *GormRoleRepository) Lookup(ctx context.Context, name string, resourceType string) (*Role, error) {
	defer goa.MeasureSince([]string{"goa", "db", "role", "lookup"}, time.Now())

	var native Role
	err := m.db.Table(m.TableName()).Joins(
		"left join resource_type on resource_type.resource_type_id = role.resource_type_id").Preload(
		"ResourceType").Where("role.name = ? and resource_type.name = ?", name, resourceType).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundError("role", name)
	}
	return &native, errs.WithStack(err)
}

func (m *GormRoleRepository) ListScopes(ctx context.Context, u *Role) ([]resource.ResourceTypeScope, error) {
	defer goa.MeasureSince([]string{"goa", "db", "role", "listscopes"}, time.Now())

	var scopes []RoleScope

	err := m.db.Where("role_id = ?", u.RoleID.String()).Preload("Scope").Find(&scopes).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}

	results := make([]resource.ResourceTypeScope, len(scopes))
	for index := 0; index < len(scopes); index++ {
		results[index] = scopes[index].Scope
	}

	return results, nil
}

func (m *GormRoleRepository) AddScope(ctx context.Context, u *Role, s *resource.ResourceTypeScope) error {
	defer goa.MeasureSince([]string{"goa", "db", "role", "addscope"}, time.Now())

	roleScope := &RoleScope{
		RoleID:  u.RoleID,
		Scope:   *s,
		ScopeID: s.ResourceTypeScopeID,
	}

	err := m.db.Create(roleScope).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"role_id":  u.RoleID,
			"scope_id": s.ResourceTypeScopeID,
			"err":      err,
		}, "unable to create the role scope")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"role_id":  u.RoleID,
		"scope_id": s.ResourceTypeScopeID,
	}, "Role scope created!")
	return nil
}
