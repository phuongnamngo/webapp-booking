import React from "react";
import { Col, Form, Row } from "react-bootstrap";
import DateTimePicker from "@/components/DateTimePicker";

export interface ModalBookingFlexibleTimePickersProps {
  enter: Date;
  leave: Date;
  disabled: boolean;
  startTimeLabel: string;
  endTimeLabel: string;
  onEnterChange: (value: Date) => void;
  onLeaveChange: (value: Date) => void;
}

function ModalBookingFlexibleTimePickers({
  enter,
  leave,
  disabled,
  startTimeLabel,
  endTimeLabel,
  onEnterChange,
  onLeaveChange,
}: ModalBookingFlexibleTimePickersProps) {
  return (
    <>
      <Form.Group as={Row} style={{ marginTop: "25px" }}>
        <Form.Label column sm="4" htmlFor="modal-enter">
          {startTimeLabel}:
        </Form.Label>
        <Col sm="8">
          <DateTimePicker
            id="modal-enter"
            value={enter}
            disabled={disabled}
            onChange={onEnterChange}
            noCalendar={true}
            enableTime={true}
            required={true}
            contained={true}
          />
        </Col>
      </Form.Group>
      <Form.Group as={Row} style={{ marginTop: "10px" }}>
        <Form.Label column sm="4" htmlFor="modal-leave">
          {endTimeLabel}:
        </Form.Label>
        <Col sm="8">
          <DateTimePicker
            id="modal-leave"
            value={leave}
            disabled={disabled}
            onChange={onLeaveChange}
            noCalendar={true}
            enableTime={true}
            required={true}
            contained={true}
          />
        </Col>
      </Form.Group>
    </>
  );
}

function propsAreEqual(
  prev: ModalBookingFlexibleTimePickersProps,
  next: ModalBookingFlexibleTimePickersProps,
): boolean {
  return (
    prev.enter.getTime() === next.enter.getTime() &&
    prev.leave.getTime() === next.leave.getTime() &&
    prev.disabled === next.disabled &&
    prev.startTimeLabel === next.startTimeLabel &&
    prev.endTimeLabel === next.endTimeLabel &&
    prev.onEnterChange === next.onEnterChange &&
    prev.onLeaveChange === next.onLeaveChange
  );
}

export default React.memo(ModalBookingFlexibleTimePickers, propsAreEqual);
