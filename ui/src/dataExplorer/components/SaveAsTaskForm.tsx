// Libraries
import React, {PureComponent, ChangeEvent} from 'react'
import {connect} from 'react-redux'

// Components
import TaskForm from 'src/tasks/components/TaskForm'

// Actions
import {saveNewScript, setTaskOption, clearTask} from 'src/tasks/actions/v2'

// Types
import {AppState, Organization} from 'src/types/v2'
import {
  TaskSchedule,
  TaskOptions,
  TaskOptionKeys,
} from 'src/utils/taskOptionsToFluxScript'

interface OwnProps {
  dismiss: () => void
}

interface DispatchProps {
  saveNewScript: typeof saveNewScript
  setTaskOption: typeof setTaskOption
  clearTask: typeof clearTask
}

interface StateProps {
  orgs: Organization[]
  taskOptions: TaskOptions
}

type Props = StateProps & OwnProps & DispatchProps

class SaveAsTaskForm extends PureComponent<Props> {
  public componentDidMount() {
    this.props.setTaskOption({
      key: 'taskScheduleType',
      value: TaskSchedule.interval,
    })
  }

  public componentWillUnmount() {
    const {clearTask} = this.props

    clearTask()
  }
  public render() {
    const {orgs, taskOptions, dismiss} = this.props

    return (
      <TaskForm
        orgs={orgs}
        taskOptions={taskOptions}
        onChangeScheduleType={this.handleChangeScheduleType}
        onChangeInput={this.handleChangeInput}
        onChangeTaskOrgID={this.handleChangeTaskOrgID}
        isInOverlay={true}
        onSubmit={this.handleSubmit}
        canSubmit={this.isFormValid}
        dismiss={dismiss}
      />
    )
  }

  private get isFormValid(): boolean {
    const {
      taskOptions: {name, cron, interval},
    } = this.props

    const hasSchedule = !!cron || !!interval
    return hasSchedule && !!name
  }

  private handleSubmit = () => {
    const {saveNewScript} = this.props
    saveNewScript()
  }

  private handleChangeTaskOrgID = (orgID: string) => {
    const {setTaskOption} = this.props

    setTaskOption({key: 'orgID', value: orgID})
  }

  private handleChangeScheduleType = (taskScheduleType: TaskSchedule) => {
    const {setTaskOption} = this.props

    setTaskOption({key: 'taskScheduleType', value: taskScheduleType})
  }

  private handleChangeInput = (e: ChangeEvent<HTMLInputElement>) => {
    const {setTaskOption} = this.props

    const key = e.target.name as TaskOptionKeys
    const value = e.target.value

    setTaskOption({key, value})
  }
}

const mstp = (state: AppState): StateProps => {
  const {orgs, tasks} = state
  return {orgs, taskOptions: tasks.taskOptions}
}

const mdtp: DispatchProps = {
  saveNewScript,
  setTaskOption,
  clearTask,
}

export default connect<StateProps, DispatchProps>(
  mstp,
  mdtp
)(SaveAsTaskForm)
