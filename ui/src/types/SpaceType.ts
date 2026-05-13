import { Entity } from "./Entity";
import Ajax from "../util/Ajax";

export type SpaceTypeBookingMode = "flexible_time" | "fixed_slots";

export class SpaceTypeSlot extends Entity {
  spaceTypeId = "";
  label = "";
  startTime = "";
  endTime = "";
  enabled = true;
  sortOrder = 0;

  serialize(): Object {
    return Object.assign(super.serialize(), {
      label: this.label,
      startTime: this.startTime,
      endTime: this.endTime,
      enabled: this.enabled,
      sortOrder: this.sortOrder,
    });
  }

  deserialize(input: any): void {
    super.deserialize(input);
    this.spaceTypeId = input.spaceTypeId || "";
    this.label = input.label || "";
    this.startTime = input.startTime || "";
    this.endTime = input.endTime || "";
    this.enabled = input.enabled !== undefined ? input.enabled : true;
    this.sortOrder = input.sortOrder || 0;
  }

  getBackendUrl(): string {
    return "";
  }
}

export default class SpaceType extends Entity {
  organizationId = "";
  name = "";
  bookingMode: SpaceTypeBookingMode = "flexible_time";
  minDurationMinutes = 0;
  enabled = true;
  slots: SpaceTypeSlot[] = [];

  serialize(): Object {
    return Object.assign(super.serialize(), {
      name: this.name,
      bookingMode: this.bookingMode,
      minDurationMinutes: this.minDurationMinutes,
      enabled: this.enabled,
    });
  }

  deserialize(input: any): void {
    super.deserialize(input);
    this.organizationId = input.organizationId || "";
    this.name = input.name || "";
    this.bookingMode = input.bookingMode || "flexible_time";
    this.minDurationMinutes = input.minDurationMinutes || 0;
    this.enabled = input.enabled !== undefined ? input.enabled : true;
    this.slots = (input.slots || []).map((item: any) => {
      const slot = new SpaceTypeSlot();
      slot.deserialize(item);
      return slot;
    });
  }

  getBackendUrl(): string {
    return "/space-type/";
  }

  async save(): Promise<SpaceType> {
    return Ajax.saveEntity(this, this.getBackendUrl()).then(() => this);
  }

  async delete(): Promise<void> {
    return Ajax.delete(this.getBackendUrl() + this.id).then(() => undefined);
  }

  async saveSlot(slot: SpaceTypeSlot): Promise<SpaceTypeSlot> {
    const url = this.getBackendUrl() + this.id + "/slot/";
    if (slot.id) {
      return Ajax.putData(url + slot.id, slot.serialize()).then(() => slot);
    }
    return Ajax.postData(url, slot.serialize()).then((result) => {
      slot.id = result.objectId;
      slot.spaceTypeId = this.id;
      return slot;
    });
  }

  async deleteSlot(slot: SpaceTypeSlot): Promise<void> {
    return Ajax.delete(
      this.getBackendUrl() + this.id + "/slot/" + slot.id,
    ).then(() => undefined);
  }

  static async get(id: string): Promise<SpaceType> {
    return Ajax.get("/space-type/" + id).then((result) => {
      const item = new SpaceType();
      item.deserialize(result.json);
      return item;
    });
  }

  static async list(): Promise<SpaceType[]> {
    return Ajax.get("/space-type/").then((result) =>
      (result.json as []).map((raw) => {
        const item = new SpaceType();
        item.deserialize(raw);
        return item;
      }),
    );
  }
}
