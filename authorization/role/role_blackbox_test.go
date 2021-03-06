package role_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-auth/authorization/resource"
	"github.com/fabric8-services/fabric8-auth/authorization/role"
	"github.com/fabric8-services/fabric8-auth/errors"
	"github.com/fabric8-services/fabric8-auth/gormtestsupport"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type roleBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo                  role.RoleRepository
	resourceTypeRepo      resource.ResourceTypeRepository
	resourceTypeScopeRepo resource.ResourceTypeScopeRepository
}

type KnownRole struct {
	ResourceTypeName string
	RoleName         string
}

var knownRoles = []KnownRole{
	{ResourceTypeName: "identity/organization", RoleName: "owner"},
}

func TestRunRoleBlackBoxTest(t *testing.T) {
	suite.Run(t, &roleBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *roleBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.DB.LogMode(true)
	s.repo = role.NewRoleRepository(s.DB)
	s.resourceTypeRepo = resource.NewResourceTypeRepository(s.DB)
	s.resourceTypeScopeRepo = resource.NewResourceTypeScopeRepository(s.DB)
}

func (s *roleBlackBoxTest) TestOKToDelete() {
	// create 2 roles, where the first one would be deleted.
	role := createAndLoadRole(s)
	createAndLoadRole(s)

	err := s.repo.Delete(s.Ctx, role.RoleID)
	assert.Nil(s.T(), err)

	// lets see how many are present.
	roles, err := s.repo.List(s.Ctx)
	require.Nil(s.T(), err, "Could not list roles")
	require.True(s.T(), len(roles) > 0)

	for _, data := range roles {
		// The role 'role' was deleted and rest were not deleted, hence we check
		// that none of the role objects returned include the one deleted.
		require.NotEqual(s.T(), role.RoleID.String(), data.RoleID.String())
	}
}

func (s *roleBlackBoxTest) TestOKToLoad() {
	createAndLoadRole(s)
}

func (s *roleBlackBoxTest) TestExistsRole() {
	t := s.T()

	t.Run("role exists", func(t *testing.T) {
		//t.Parallel()
		role := createAndLoadRole(s)
		// when
		_, err := s.repo.CheckExists(s.Ctx, role.RoleID.String())
		// then
		require.Nil(t, err)
	})

	t.Run("role doesn't exist", func(t *testing.T) {
		//t.Parallel()
		// Check not existing
		_, err := s.repo.CheckExists(s.Ctx, uuid.NewV4().String())
		// then
		require.IsType(s.T(), errors.NotFoundError{}, err)
	})
}

func (s *roleBlackBoxTest) TestOKToSave() {
	role := createAndLoadRole(s)

	role.Name = "newRoleNameTestType"
	err := s.repo.Save(s.Ctx, role)
	require.Nil(s.T(), err, "Could not update role")

	updatedRole, err := s.repo.Load(s.Ctx, role.RoleID)
	require.Nil(s.T(), err, "Could not load role")
	assert.Equal(s.T(), role.Name, updatedRole.Name)
}

func createAndLoadRole(s *roleBlackBoxTest) *role.Role {

	resourceType, err := s.resourceTypeRepo.Lookup(s.Ctx, "openshift.io/resource/area")
	require.Nil(s.T(), err, "Could not create resource type")

	role := &role.Role{
		RoleID:         uuid.NewV4(),
		ResourceType:   *resourceType,
		ResourceTypeID: resourceType.ResourceTypeID,
		Name:           "role_blackbox_test_admin" + uuid.NewV4().String(),
		//Scopes:         []resource.ResourceTypeScope{*resourceTypeScope},
	}

	err = s.repo.Create(s.Ctx, role)
	require.Nil(s.T(), err, "Could not create role")

	createdRole, err := s.repo.Load(s.Ctx, role.RoleID)
	require.Nil(s.T(), err, "Could not load role")
	require.Equal(s.T(), role.Name, createdRole.Name)
	require.Equal(s.T(), role.ResourceTypeID, createdRole.ResourceTypeID)

	return createdRole
}

func (s *roleBlackBoxTest) TestKnownRolesExist() {
	t := s.T()

	t.Run("role exists", func(t *testing.T) {

		for _, r := range knownRoles {
			_, err := s.repo.Lookup(s.Ctx, r.RoleName, r.ResourceTypeName)
			// then
			require.Nil(t, err)
		}
	})
}
