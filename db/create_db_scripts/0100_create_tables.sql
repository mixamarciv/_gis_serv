


CREATE TABLE TASK (
    IDC           VARCHAR(36),
    IN_FILE       VARCHAR(1000),
    FHASH         VARCHAR(100),
    SENDERID      VARCHAR(36),
    messageguid   VARCHAR(36),
    huisver       VARCHAR(100),
    OUT_FILE      VARCHAR(1000),
    datareq       VARCHAR(7000),
    DATE_CREATE   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    DATE_UPDATE   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    DATE_RUN      TIMESTAMP,
    DATE_END      TIMESTAMP,
    CDATE_RUN     VARCHAR(50),
    CDATE_END     VARCHAR(50),
    STATUS        SMALLINT     -- 0 - prepare, 1 - send, 2 - wait, 3 - end
);



CREATE UNIQUE INDEX TASK_IDX1 ON TASK (FHASH);
CREATE UNIQUE INDEX TASK_IDX2 ON TASK (messageguid, DATE_END, DATE_CREATE);


SET TERM ^ ;


/* Trigger: TASK_BI0 */
CREATE TRIGGER TASK_BI0 FOR TASK
ACTIVE BEFORE INSERT POSITION 0
AS
begin
  new.date_create = current_timestamp;
  IF( new.idc IS NULL ) THEN 
  BEGIN
  	new.idc = UUID_TO_CHAR( GEN_UUID() );
  END 

  IF( new.messageguid IS NULL  ) THEN new.messageguid = '';
  IF( new.CDATE_RUN IS NULL    ) THEN new.CDATE_RUN = '';
  IF( new.CDATE_END IS NULL    ) THEN new.CDATE_END = '';
  IF( new.OUT_FILE IS NULL     ) THEN new.OUT_FILE = '';
  IF( new.SENDERID IS NULL     ) THEN new.SENDERID = '';
  IF( new.huisver IS NULL      ) THEN new.huisver = '';
  IF( new.datareq IS NULL      ) THEN new.datareq = '';

end
^


/* Trigger: TASK_BU0 */
CREATE TRIGGER TASK_BU0 FOR TASK
ACTIVE BEFORE UPDATE POSITION 0
AS
begin
  new.date_create = old.date_create;
  new.date_update = current_timestamp;
end
^

SET TERM ; ^

COMMIT WORK;
