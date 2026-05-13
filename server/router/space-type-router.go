package router

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	. "github.com/seatsurfing/seatsurfing/server/repository"
)

type SpaceTypeRouter struct {
}

type CreateSpaceTypeRequest struct {
	Name               string `json:"name" validate:"required,max=128"`
	BookingMode        string `json:"bookingMode" validate:"required"`
	MinDurationMinutes int    `json:"minDurationMinutes"`
	Enabled            bool   `json:"enabled"`
}

type CreateSpaceTypeSlotRequest struct {
	Label     string `json:"label" validate:"required,max=128"`
	StartTime string `json:"startTime" validate:"required"`
	EndTime   string `json:"endTime" validate:"required"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sortOrder"`
}

type GetSpaceTypeResponse struct {
	ID             string                      `json:"id"`
	OrganizationID string                      `json:"organizationId"`
	Slots          []*GetSpaceTypeSlotResponse `json:"slots"`
	CreateSpaceTypeRequest
}

type GetSpaceTypeSlotResponse struct {
	ID          string `json:"id"`
	SpaceTypeID string `json:"spaceTypeId"`
	CreateSpaceTypeSlotRequest
}

func (router *SpaceTypeRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/{id}/slot/{slotId}", router.updateSlot).Methods("PUT")
	s.HandleFunc("/{id}/slot/{slotId}", router.deleteSlot).Methods("DELETE")
	s.HandleFunc("/{id}/slot/", router.createSlot).Methods("POST")
	s.HandleFunc("/{id}", router.getOne).Methods("GET")
	s.HandleFunc("/{id}", router.update).Methods("PUT")
	s.HandleFunc("/{id}", router.delete).Methods("DELETE")
	s.HandleFunc("/", router.create).Methods("POST")
	s.HandleFunc("/", router.getAll).Methods("GET")
}

func (router *SpaceTypeRouter) getOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetSpaceTypeRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !CanAccessOrg(user, e.OrganizationID) {
		SendForbidden(w)
		return
	}
	SendJSON(w, router.copyToRestModel(e))
}

func (router *SpaceTypeRouter) getAll(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	list, err := GetSpaceTypeRepository().GetAll(user.OrganizationID)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetSpaceTypeResponse{}
	for _, e := range list {
		res = append(res, router.copyToRestModel(e))
	}
	SendJSON(w, res)
}

func (router *SpaceTypeRouter) create(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceTypeRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, user.OrganizationID) {
		SendForbidden(w)
		return
	}
	e := router.copyFromRestModel(&m)
	e.OrganizationID = user.OrganizationID
	if err := GetSpaceTypeRepository().Create(e); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	SendCreated(w, e.ID)
}

func (router *SpaceTypeRouter) update(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceTypeRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	vars := mux.Vars(r)
	existing, err := GetSpaceTypeRepository().GetOne(vars["id"])
	if err != nil {
		SendBadRequest(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, existing.OrganizationID) {
		SendForbidden(w)
		return
	}
	e := router.copyFromRestModel(&m)
	e.ID = existing.ID
	e.OrganizationID = existing.OrganizationID
	if err := GetSpaceTypeRepository().Update(e); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceTypeRouter) delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetSpaceTypeRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, e.OrganizationID) {
		SendForbidden(w)
		return
	}
	if err := GetSpaceTypeRepository().Delete(e.ID); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceTypeRouter) createSlot(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceTypeSlotRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	spaceType, ok := router.getSpaceTypeForAdmin(w, r)
	if !ok {
		return
	}
	slot := router.copySlotFromRestModel(&m)
	slot.SpaceTypeID = spaceType.ID
	if err := GetSpaceTypeRepository().CreateSlot(slot); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	SendCreated(w, slot.ID)
}

func (router *SpaceTypeRouter) updateSlot(w http.ResponseWriter, r *http.Request) {
	var m CreateSpaceTypeSlotRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	spaceType, ok := router.getSpaceTypeForAdmin(w, r)
	if !ok {
		return
	}
	slot, ok := router.getSlotForSpaceType(w, r, spaceType.ID)
	if !ok {
		return
	}
	slot.Label = m.Label
	slot.StartTime = m.StartTime
	slot.EndTime = m.EndTime
	slot.Enabled = m.Enabled
	slot.SortOrder = m.SortOrder
	if err := GetSpaceTypeRepository().UpdateSlot(slot); err != nil {
		log.Println(err)
		SendBadRequest(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceTypeRouter) deleteSlot(w http.ResponseWriter, r *http.Request) {
	spaceType, ok := router.getSpaceTypeForAdmin(w, r)
	if !ok {
		return
	}
	slot, ok := router.getSlotForSpaceType(w, r, spaceType.ID)
	if !ok {
		return
	}
	if err := GetSpaceTypeRepository().DeleteSlot(slot.ID); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *SpaceTypeRouter) getSpaceTypeForAdmin(w http.ResponseWriter, r *http.Request) (*SpaceType, bool) {
	vars := mux.Vars(r)
	spaceType, err := GetSpaceTypeRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return nil, false
	}
	user := GetRequestUser(r)
	if !CanSpaceAdminOrg(user, spaceType.OrganizationID) {
		SendForbidden(w)
		return nil, false
	}
	return spaceType, true
}

func (router *SpaceTypeRouter) getSlotForSpaceType(w http.ResponseWriter, r *http.Request, spaceTypeID string) (*SpaceTypeSlot, bool) {
	vars := mux.Vars(r)
	slots, err := GetSpaceTypeRepository().GetSlots(spaceTypeID, false)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return nil, false
	}
	for _, slot := range slots {
		if slot.ID == vars["slotId"] {
			return slot, true
		}
	}
	SendNotFound(w)
	return nil, false
}

func (router *SpaceTypeRouter) copyFromRestModel(m *CreateSpaceTypeRequest) *SpaceType {
	e := &SpaceType{}
	e.Name = m.Name
	e.BookingMode = m.BookingMode
	e.MinDurationMinutes = m.MinDurationMinutes
	e.Enabled = m.Enabled
	return e
}

func (router *SpaceTypeRouter) copyToRestModel(e *SpaceType) *GetSpaceTypeResponse {
	m := &GetSpaceTypeResponse{}
	m.ID = e.ID
	m.OrganizationID = e.OrganizationID
	m.Name = e.Name
	m.BookingMode = e.BookingMode
	m.MinDurationMinutes = e.MinDurationMinutes
	m.Enabled = e.Enabled
	for _, slot := range e.Slots {
		m.Slots = append(m.Slots, router.copySlotToRestModel(slot))
	}
	return m
}

func (router *SpaceTypeRouter) copySlotFromRestModel(m *CreateSpaceTypeSlotRequest) *SpaceTypeSlot {
	e := &SpaceTypeSlot{}
	e.Label = m.Label
	e.StartTime = m.StartTime
	e.EndTime = m.EndTime
	e.Enabled = m.Enabled
	e.SortOrder = m.SortOrder
	return e
}

func (router *SpaceTypeRouter) copySlotToRestModel(e *SpaceTypeSlot) *GetSpaceTypeSlotResponse {
	m := &GetSpaceTypeSlotResponse{}
	m.ID = e.ID
	m.SpaceTypeID = e.SpaceTypeID
	m.Label = e.Label
	m.StartTime = e.StartTime
	m.EndTime = e.EndTime
	m.Enabled = e.Enabled
	m.SortOrder = e.SortOrder
	return m
}
