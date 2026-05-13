import React from "react";
import { Table } from "react-bootstrap";
import { Plus as IconPlus } from "react-feather";
import FullLayout from "@/components/FullLayout";
import { NextRouter } from "next/router";
import Link from "next/link";
import Loading from "@/components/Loading";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import SpaceType from "@/types/SpaceType";
import Ajax from "@/util/Ajax";
import RedirectUtil from "@/util/RedirectUtil";
import RendererUtils from "@/util/RendererUtils";
import Navigation from "@/util/Navigation";
import AdminBooleanState from "@/components/AdminBooleanState";

interface State {
  selectedItem: string;
  loading: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class SeatTypes extends React.Component<Props, State> {
  data: SpaceType[] = [];

  constructor(props: any) {
    super(props);
    this.state = {
      selectedItem: "",
      loading: true,
    };
  }

  componentDidMount = () => {
    if (!Ajax.hasAccessToken()) {
      RedirectUtil.toLogin(this.props.router);
      return;
    }
    this.loadItems();
  };

  loadItems = () => {
    SpaceType.list().then((list) => {
      this.data = list;
      this.setState({ loading: false });
    });
  };

  getBookingModeLabel = (spaceType: SpaceType) => {
    if (spaceType.bookingMode === "fixed_slots") {
      return this.props.t("fixedSlots");
    }
    return this.props.t("flexibleTime");
  };

  renderItem = (spaceType: SpaceType) => (
    <tr
      key={spaceType.id}
      onClick={() => this.setState({ selectedItem: spaceType.id })}
    >
      <td>{spaceType.name}</td>
      <td>{this.getBookingModeLabel(spaceType)}</td>
      <td>
        <AdminBooleanState value={spaceType.enabled} />
      </td>
    </tr>
  );

  render() {
    if (this.state.selectedItem) {
      this.props.router.push(
        Navigation.adminSeatTypeDetails(this.state.selectedItem),
      );
      return <></>;
    }

    const buttons = (
      <Link
        href={Navigation.adminSeatTypeDetails("add")}
        className="btn btn-sm btn-outline-secondary"
      >
        <IconPlus className="feather" /> {this.props.t("add")}
      </Link>
    );

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("seatTypes")} buttons={buttons}>
          <Loading />
        </FullLayout>
      );
    }

    const rows = this.data.map((item) => this.renderItem(item));
    if (rows.length === 0) {
      return (
        <FullLayout headline={this.props.t("seatTypes")} buttons={buttons}>
          <p>{this.props.t("noRecords")}</p>
        </FullLayout>
      );
    }

    return (
      <FullLayout headline={this.props.t("seatTypes")} buttons={buttons}>
        <Table striped={true} hover={true} className="clickable-table">
          <thead>
            <tr>
              <th>{this.props.t("name")}</th>
              <th>{this.props.t("bookingMode")}</th>
              <th>{this.props.t("enabled")}</th>
            </tr>
          </thead>
          <tbody>{rows}</tbody>
        </Table>
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(SeatTypes as any));
