import { QueryProps } from 'container/QueryBuilder/components/Query/Query.interfaces';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { IBuilderQueryForm } from 'types/api/queryBuilder/queryBuilderData';
import { DataSource } from 'types/common/queryBuilder';

type UseQueryOperationsParams = Pick<
	QueryProps,
	'index' | 'panelType' | 'query'
>;

export type HandleChangeQueryData = <
	Key extends keyof IBuilderQueryForm,
	Value extends IBuilderQueryForm[Key]
>(
	key: Key,
	value: Value,
) => void;

export type UseQueryOperations = (
	params: UseQueryOperationsParams,
) => {
	isMetricsDataSource: boolean;
	operators: string[];
	handleChangeOperator: (value: string) => void;
	handleChangeAggregatorAttribute: (value: BaseAutocompleteData) => void;
	handleChangeDataSource: (newSource: DataSource) => void;
	handleDeleteQuery: () => void;
	handleChangeQueryData: HandleChangeQueryData;
};
