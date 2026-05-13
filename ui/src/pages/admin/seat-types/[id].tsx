import React from "react";
import { Form, Col, Row, Button, Alert, Table } from "react-bootstrap";
import {
  ChevronLeft as IconBack,
  Save as IconSave,
  Trash2 as IconDelete,
  Plus as IconPlus,
} from "react-feather";
import { NextRouter } from "next/router";
import FullLayout from "@/components/FullLayout";
import Loading from "@/components/Loading";
import Link from "next/link";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import SpaceType, {
  SpaceTypeBookingMode,
  SpaceTypeSlot,
} from "@/types/SpaceType";
import Ajax from "@/util/Ajax";
import RedirectUtil from "@/util/RedirectUtil";
import Navigation from "@/util/Navigation";

interface State {
  loading: boolean;
  saved: boolean;
  error: boolean;
  goBack: boolean;
  name: string;
  bookingMode: SpaceTypeBookingMode;
  minDurationMinutes: number;
  enabled: boolean;
  slots: SpaceTypeSlot[];
  deletedSlots: SpaceTypeSlot[];
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class EditSeatType extends React.Component<Props, State> {
  entity: SpaceType = new SpaceType();

  constructor(props: any) {
    super(props);
    this.state = {
      loading: true,
      saved: false,
      error: false,
      goBack: false,
      name: "",
      bookingMode: "flexible_time",
      minDurationMinutes: 30,
      enabled: true,
      slots: [],
      deletedSlots: [],
    };
  }

  componentDidMount = () => {
    if (!Ajax.hasAccessToken()) {
      RedirectUtil.toLogin(this.props.router);
      return;
    }
    this.loadData();
  };

  loadData = () => {
    const { id } = this.props.router.query;
    if (id && typeof id === "string" && id !== "add") {
      SpaceType.get(id).then((spaceType) => {
        this.entity = spaceType;
        this.setState({
          name: spaceType.name,
          bookingMode: spaceType.bookingMode,
          minDurationMinutes: spaceType.minDurationMinutes,
          enabled: spaceType.enabled,
          slots: spaceType.slots,
          loading: false,
        });
      });
    } else {
      this.setState({ loading: false });
    }
  };

  addSlot = () => {
    const slot = new SpaceTypeSlot();
    slot.enabled = true;
    slot.sortOrder = this.state.slots.length + 1;
    this.setState({ slots: [...this.state.slots, slot] });
  };

  updateSlot = (index: number, changes: Partial<SpaceTypeSlot>) => {
    const slots = [...this.state.slots];
    slots[index] = Object.assign(new SpaceTypeSlot(), slots[index], changes);
    this.setState({ slots });
  };

  removeSlot = (index: number) => {
    const slots = [...this.state.slots];
    const [slot] = slots.splice(index, 1);
    const deletedSlots = slot.id
      ? [...this.state.deletedSlots, slot]
      : this.state.deletedSlots;
    this.setState({ slots, deletedSlots });
  };

  onSubmit = (e: any) => {
    e.preventDefault();
    this.setState({ error: false, saved: false });
    this.entity.name = this.state.name;
    this.entity.bookingMode = this.state.bookingMode;
    this.entity.minDurationMinutes = Number(this.state.minDurationMinutes);
    this.entity.enabled = this.state.enabled;
    this.entity
      .save()
      .then(() => {
        const slotTasks = [
          ...this.state.deletedSlots.map((slot) =>
            this.entity.deleteSlot(slot),
          ),
          ...this.state.slots.map((slot) => this.entity.saveSlot(slot)),
        ];
        return Promise.all(slotTasks);
      })
      .then(() => {
        this.props.router.push(Navigation.adminSeatTypeDetails(this.entity.id));
        this.setState({ saved: true, deletedSlots: [] });
        this.loadData();
      })
      .catch(() => this.setState({ error: true }));
  };

  deleteItem = () => {
    if (window.confirm(this.props.t("confirmDeleteSeatType"))) {
      this.entity
        .delete()
        .then(() => this.setState({ goBack: true }))
        .catch(() => this.setState({ error: true }));
    }
  };

  renderSlots = () => {
    if (this.state.bookingMode !== "fixed_slots") {
      return <></>;
    }
    return (
      <>
        <h3>{this.props.t("timeSlots")}</h3>
        <Table striped={true}>
          <thead>
            <tr>
              <th>{this.props.t("name")}</th>
              <th>{this.props.t("startTime")}</th>
              <th>{this.props.t("endTime")}</th>
              <th>{this.props.t("enabled")}</th>
              <th>{this.props.t("sortOrder")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {this.state.slots.map((slot, index) => (
              <tr key={slot.id || index}>
                <td>
                  <Form.Control
                    value={slot.label}
                    onChange={(e: any) =>
                      this.updateSlot(index, { label: e.target.value })
                    }
                    required={true}
                  />
                </td>
                <td>
                  <Form.Control
                    type="time"
                    value={slot.startTime}
                    onChange={(e: any) =>
                      this.updateSlot(index, { startTime: e.target.value })
                    }
                    required={true}
                  />
                </td>
                <td>
                  <Form.Control
                    type="time"
                    value={slot.endTime}
                    onChange={(e: any) =>
                      this.updateSlot(index, { endTime: e.target.value })
                    }
                    required={true}
                  />
                </td>
                <td>
                  <Form.Check
                    checked={slot.enabled}
                    onChange={(e: any) =>
                      this.updateSlot(index, { enabled: e.target.checked })
                    }
                  />
                </td>
                <td>
                  <Form.Control
                    type="number"
                    value={slot.sortOrder}
                    onChange={(e: any) =>
                      this.updateSlot(index, {
                        sortOrder: Number(e.target.value),
                      })
                    }
                  />
                </td>
                <td>
                  <Button
                    variant="outline-secondary"
                    className="btn-sm"
                    onClick={() => this.removeSlot(index)}
                  >
                    <IconDelete className="feather" />
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
        <Button
          variant="outline-secondary"
          className="btn-sm"
          onClick={this.addSlot}
        >
          <IconPlus className="feather" /> {this.props.t("add")}
        </Button>
      </>
    );
  };

  render() {
    if (this.state.goBack) {
      this.props.router.push(Navigation.adminSeatTypes());
      return <></>;
    }

    const backButton = (
      <Link
        href={Navigation.adminSeatTypes()}
        className="btn btn-sm btn-outline-secondary"
      >
        <IconBack className="feather" /> {this.props.t("back")}
      </Link>
    );
    const buttonSave = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        type="submit"
        form="form"
      >
        <IconSave className="feather" /> {this.props.t("save")}
      </Button>
    );
    const buttonDelete = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        onClick={this.deleteItem}
      >
        <IconDelete className="feather" /> {this.props.t("delete")}
      </Button>
    );
    const buttons = this.entity.id ? (
      <>
        {backButton} {buttonDelete} {buttonSave}
      </>
    ) : (
      <>
        {backButton} {buttonSave}
      </>
    );

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("editSeatType")} buttons={buttons}>
          <Loading />
        </FullLayout>
      );
    }

    let hint = <></>;
    if (this.state.saved) {
      hint = <Alert variant="success">{this.props.t("entryUpdated")}</Alert>;
    } else if (this.state.error) {
      hint = <Alert variant="danger">{this.props.t("errorSave")}</Alert>;
    }

    return (
      <FullLayout headline={this.props.t("editSeatType")} buttons={buttons}>
        <Form onSubmit={this.onSubmit} id="form">
          {hint}
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("name")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                value={this.state.name}
                onChange={(e: any) => this.setState({ name: e.target.value })}
                required={true}
                autoFocus={true}
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("bookingMode")}
            </Form.Label>
            <Col sm="4">
              <Form.Select
                value={this.state.bookingMode}
                onChange={(e: any) =>
                  this.setState({ bookingMode: e.target.value })
                }
              >
                <option value="flexible_time">
                  {this.props.t("flexibleTime")}
                </option>
                <option value="fixed_slots">
                  {this.props.t("fixedSlots")}
                </option>
              </Form.Select>
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("minimumDurationMinutes")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="number"
                min={0}
                value={this.state.minDurationMinutes}
                onChange={(e: any) =>
                  this.setState({
                    minDurationMinutes: Number(e.target.value),
                  })
                }
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("enabled")}
            </Form.Label>
            <Col sm="4">
              <Form.Check
                checked={this.state.enabled}
                onChange={(e: any) =>
                  this.setState({ enabled: e.target.checked })
                }
              />
            </Col>
          </Form.Group>
          {this.renderSlots()}
        </Form>
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(EditSeatType as any));
